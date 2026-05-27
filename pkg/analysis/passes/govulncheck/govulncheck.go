package govulncheck

import (
	"bytes"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	govulncheckNotInstalled  = &analysis.Rule{Name: "govulncheck-not-installed", Severity: analysis.Warning}
	govulncheckScanFailed    = &analysis.Rule{Name: "govulncheck-scan-failed", Severity: analysis.Error}
	govulncheckIssueFound    = &analysis.Rule{Name: "govulncheck-issue-found", Severity: analysis.Warning}
	govulncheckNoIssuesFound = &analysis.Rule{Name: "govulncheck-no-issues-found", Severity: analysis.OK}
)

var Analyzer = &analysis.Analyzer{
	Name:     "govulncheck",
	Requires: []*analysis.Analyzer{archive.Analyzer, nestedmetadata.Analyzer, sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{govulncheckNotInstalled, govulncheckScanFailed, govulncheckIssueFound, govulncheckNoIssuesFound},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Go Vulnerability Checker",
		Description:  "Scans Go backend source and plugin backend binaries for known vulnerabilities (govulncheck).",
		Dependencies: "[govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck), `sourceCodeUri` for source scans",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	govulncheckBin, err := exec.LookPath("govulncheck")
	if err != nil {
		logme.Debugln("govulncheck not installed, skipping govulncheck analysis")
		if govulncheckNotInstalled.ReportAll {
			pass.ReportResult(
				pass.AnalyzerName,
				govulncheckNotInstalled,
				"govulncheck not installed",
				"Skipping govulncheck analysis",
			)
		}
		return nil, nil
	}

	scansPerformed := 0
	scanFailures := 0
	findingsReported := 0

	// Source scan: mirrors gosec's pattern of silently skipping when source
	// isn't provided. Scans whatever go.mod modules exist under the source
	// tree without gating on backend status.
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if ok && sourceCodeDir != "" {
		moduleDirs, err := goModuleDirs(sourceCodeDir)
		if err != nil {
			return nil, err
		}
		sourceFindings := make(map[string]struct{})
		for _, moduleDir := range moduleDirs {
			stdout, ok, failureDetail, err := runGovulncheckJSON(govulncheckBin, moduleDir, moduleDir, "-json", "./...")
			if err != nil {
				return nil, err
			}
			if !ok {
				scanFailures++
				pass.ReportResult(
					pass.AnalyzerName,
					govulncheckScanFailed,
					"govulncheck source scan failed",
					failureDetail,
				)
			} else {
				scansPerformed++
				osvIDs, err := parseCalledFindings(bytes.NewReader(stdout))
				if err != nil {
					logme.Errorln("Error parsing govulncheck source output", "error", err)
					return nil, err
				}
				for id := range osvIDs {
					sourceFindings[id] = struct{}{}
				}
			}
		}
		findingsReported += len(sourceFindings)
		reportSourceFindings(pass, sourceFindings)
	}

	binaryFindings := make(map[string]map[string]struct{})
	binaryPaths, err := backendBinaries(pass)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			govulncheckScanFailed,
			"govulncheck binary scan failed",
			err.Error(),
		)
		return nil, nil
	}
	for _, binaryPath := range binaryPaths {
		stdout, ok, failureDetail, err := runGovulncheckJSON(govulncheckBin, "", filepath.Base(binaryPath), "-mode=binary", "-json", binaryPath)
		if err != nil {
			return nil, err
		}
		if !ok {
			scanFailures++
			pass.ReportResult(
				pass.AnalyzerName,
				govulncheckScanFailed,
				fmt.Sprintf("govulncheck binary scan failed for %s", filepath.Base(binaryPath)),
				failureDetail,
			)
			continue
		}
		scansPerformed++
		osvIDs, err := parseAllFindings(bytes.NewReader(stdout))
		if err != nil {
			logme.Errorln("Error parsing govulncheck binary output", "error", err)
			return nil, err
		}
		for id := range osvIDs {
			if binaryFindings[id] == nil {
				binaryFindings[id] = make(map[string]struct{})
			}
			binaryFindings[id][filepath.Base(binaryPath)] = struct{}{}
		}
	}
	findingsReported += len(binaryFindings)
	reportBinaryFindings(pass, binaryFindings)

	if scansPerformed > 0 && scanFailures == 0 && findingsReported == 0 && govulncheckNoIssuesFound.ReportAll {
		pass.ReportResult(
			pass.AnalyzerName,
			govulncheckNoIssuesFound,
			"govulncheck reports no vulnerabilities",
			"",
		)
	}

	return nil, nil
}

func runGovulncheckJSON(govulncheckBin, dir, target string, args ...string) ([]byte, bool, string, error) {
	cmd := exec.Command(govulncheckBin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// Some govulncheck versions exit 3 when vulnerabilities are found.
		// Other non-zero exits are scanner failures, such as package loading
		// errors or unsupported binary formats.
		exitErr, isExit := err.(*exec.ExitError)
		if !isExit {
			logme.ErrorF("Error running govulncheck for %s: %v (stderr: %s)", target, err, stderr.String())
			return nil, false, "", err
		}
		if exitErr.ExitCode() != 3 {
			logme.DebugFln("govulncheck scan failed for %s: %v (stderr: %s)", target, err, stderr.String())
			return nil, false, scanFailureDetail(target, stderr.String(), err), nil
		}
	}
	return stdout.Bytes(), true, "", nil
}

// parseCalledFindings decodes the govulncheck `-json` NDJSON stream and
// returns the set of OSV IDs whose Finding contains a call-site frame
// (i.e. the vulnerable symbol is reachable from user code, not merely
// present in a transitive dependency).
func parseCalledFindings(r io.Reader) (map[string]struct{}, error) {
	dec := json.NewDecoder(r)
	called := make(map[string]struct{})
	for {
		var msg Message
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if msg.Finding == nil || msg.Finding.OSV == "" {
			continue
		}
		if isCalled(msg.Finding) {
			called[msg.Finding.OSV] = struct{}{}
		}
	}
	return called, nil
}

func parseAllFindings(r io.Reader) (map[string]struct{}, error) {
	dec := json.NewDecoder(r)
	found := make(map[string]struct{})
	for {
		var msg Message
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if msg.Finding == nil || msg.Finding.OSV == "" {
			continue
		}
		found[msg.Finding.OSV] = struct{}{}
	}
	return found, nil
}

// isCalled returns true if the Finding's call trace includes a concrete
// call-site frame. govulncheck emits findings at three levels: module,
// package, and symbol/called; only symbol/called findings have a Position.
func isCalled(f *Finding) bool {
	for _, frame := range f.Trace {
		if frame.Position != nil && frame.Position.Filename != "" {
			return true
		}
	}
	return false
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func scanFailureDetail(target, stderr string, err error) string {
	detail := strings.TrimSpace(stderr)
	if detail == "" && err != nil {
		detail = err.Error()
	}
	if detail == "" {
		detail = "govulncheck exited unsuccessfully without details."
	}
	if target != "" {
		detail = fmt.Sprintf("%s: %s", target, detail)
	}
	const maxDetailLen = 1000
	if len(detail) > maxDetailLen {
		return detail[:maxDetailLen] + "..."
	}
	return detail
}

func goModuleDirs(sourceCodeDir string) ([]string, error) {
	var moduleDirs []string
	err := filepath.WalkDir(sourceCodeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "dist", "node_modules", "vendor":
				if path != sourceCodeDir {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if d.Name() == "go.mod" {
			moduleDirs = append(moduleDirs, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error finding Go modules in %s: %w", sourceCodeDir, err)
	}
	sort.Strings(moduleDirs)
	return moduleDirs, nil
}

func backendBinaries(pass *analysis.Pass) ([]string, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok || archiveDir == "" {
		return nil, nil
	}
	metadatamap, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		return nil, nil
	}

	var binaries []string
	for pluginJSONPath, data := range metadatamap {
		if data.Executable == "" {
			continue
		}
		relativeTo := filepath.Join(archiveDir, filepath.Dir(pluginJSONPath))
		executable := data.Executable
		executableParentDir := filepath.Join(relativeTo, filepath.Dir(executable))
		executableName := filepath.Base(executable)

		entries, err := os.ReadDir(executableParentDir)
		if err != nil {
			return nil, fmt.Errorf("error reading backend binaries for %s: %w", pluginJSONPath, err)
		}
		var candidateErrors []string
		validForExecutable := 0
		for _, entry := range entries {
			if entry.IsDir() || !entry.Type().IsRegular() {
				continue
			}
			if !strings.HasPrefix(entry.Name(), executableName) {
				continue
			}
			path := filepath.Join(executableParentDir, entry.Name())
			ok, err := isGoBinaryCandidate(path)
			if err != nil {
				candidateErrors = append(candidateErrors, err.Error())
				continue
			}
			if !ok {
				continue
			}
			validForExecutable++
			binaries = append(binaries, path)
		}
		if validForExecutable == 0 && len(candidateErrors) > 0 {
			return nil, fmt.Errorf("no scannable Go backend binary found for %s: %s", pluginJSONPath, strings.Join(candidateErrors, "; "))
		}
	}
	sort.Strings(binaries)
	return binaries, nil
}

func isGoBinaryCandidate(path string) (bool, error) {
	_, err := buildinfo.ReadFile(path)
	if err == nil {
		return true, nil
	}
	return false, fmt.Errorf("%s is not a Go binary: %w", path, err)
}

func reportSourceFindings(pass *analysis.Pass, osvIDs map[string]struct{}) {
	if len(osvIDs) == 0 {
		return
	}
	ids := sortedKeys(osvIDs)
	pass.ReportResult(
		pass.AnalyzerName,
		govulncheckIssueFound,
		fmt.Sprintf("govulncheck source scan reports %d reachable vulnerabilit%s", len(ids), pluralY(len(ids))),
		fmt.Sprintf(
			"Run govulncheck https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck in your plugin source to see details. Reachable OSV IDs: %s",
			strings.Join(ids, ", "),
		),
	)
}

func reportBinaryFindings(pass *analysis.Pass, binaryFindings map[string]map[string]struct{}) {
	if len(binaryFindings) == 0 {
		return
	}
	ids := make([]string, 0, len(binaryFindings))
	for id := range binaryFindings {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%s (%s)", id, strings.Join(sortedKeys(binaryFindings[id]), ", ")))
	}

	pass.ReportResult(
		pass.AnalyzerName,
		govulncheckIssueFound,
		fmt.Sprintf("govulncheck binary scan reports %d vulnerabilit%s", len(ids), pluralY(len(ids))),
		"Detected OSV IDs in backend binaries: "+strings.Join(parts, "; "),
	)
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}

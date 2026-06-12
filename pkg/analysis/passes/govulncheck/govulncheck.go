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

	"golang.org/x/mod/semver"

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

type vulnInfo struct {
	id           string
	summary      string
	module       string
	fixedVersion string
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
			// Report as a diagnostic instead of returning an error: returning
			// an error here aborts the whole validator and skips every other
			// analyzer. Skip the source scan and continue with binary scans.
			scanFailures++
			pass.ReportResult(
				pass.AnalyzerName,
				govulncheckScanFailed,
				"govulncheck source scan failed",
				scanFailureDetail(sourceCodeDir, "", err),
			)
			moduleDirs = nil
		}
		sourceFindings := make(map[string]*vulnInfo)
		for _, moduleDir := range moduleDirs {
			stdout, ok, failureDetail, err := runGovulncheckJSON(govulncheckBin, moduleDir, moduleDir, "-json", "./...")
			if err != nil {
				scanFailures++
				pass.ReportResult(
					pass.AnalyzerName,
					govulncheckScanFailed,
					"govulncheck source scan failed",
					scanFailureDetail(moduleDir, "", err),
				)
				continue
			}
			if !ok {
				scanFailures++
				pass.ReportResult(
					pass.AnalyzerName,
					govulncheckScanFailed,
					"govulncheck source scan failed",
					failureDetail,
				)
				continue
			}
			scansPerformed++
			vulns, err := parseCalledFindings(bytes.NewReader(stdout))
			if err != nil {
				logme.Errorln("Error parsing govulncheck source output", "error", err)
				scanFailures++
				pass.ReportResult(
					pass.AnalyzerName,
					govulncheckScanFailed,
					"govulncheck source scan failed",
					scanFailureDetail(moduleDir, "", err),
				)
				continue
			}
			for id, info := range vulns {
				if sourceFindings[id] == nil {
					sourceFindings[id] = info
				}
			}
		}
		findingsReported += len(sourceFindings)
		reportSourceFindings(pass, sourceFindings)
	}

	binaryFindings := make(map[string]*vulnInfo)
	binaryPaths, err := getBackendBinaries(pass)
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
			scanFailures++
			pass.ReportResult(
				pass.AnalyzerName,
				govulncheckScanFailed,
				fmt.Sprintf("govulncheck binary scan failed for %s", filepath.Base(binaryPath)),
				scanFailureDetail(filepath.Base(binaryPath), "", err),
			)
			continue
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
		vulns, err := parseAllFindings(bytes.NewReader(stdout))
		if err != nil {
			logme.Errorln("Error parsing govulncheck binary output", "error", err)
			scanFailures++
			pass.ReportResult(
				pass.AnalyzerName,
				govulncheckScanFailed,
				fmt.Sprintf("govulncheck binary scan failed for %s", filepath.Base(binaryPath)),
				scanFailureDetail(filepath.Base(binaryPath), "", err),
			)
			continue
		}
		for id, info := range vulns {
			if binaryFindings[id] == nil {
				binaryFindings[id] = info
			}
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
// returns vulns whose Finding contains a call-site frame (i.e. the vulnerable
// symbol is reachable from user code, not merely present in a transitive dep).
func parseCalledFindings(r io.Reader) (map[string]*vulnInfo, error) {
	dec := json.NewDecoder(r)
	summaries := make(map[string]string)
	called := make(map[string]*vulnInfo)
	for {
		var msg Message
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if msg.OSV != nil && msg.OSV.ID != "" {
			summaries[msg.OSV.ID] = msg.OSV.Summary
		}
		if msg.Finding == nil || msg.Finding.OSV == "" {
			continue
		}
		if isCalled(msg.Finding) {
			id := msg.Finding.OSV
			if called[id] == nil {
				called[id] = &vulnInfo{id: id}
			}
			if semver.Compare(msg.Finding.FixedVersion, called[id].fixedVersion) > 0 {
				called[id].fixedVersion = msg.Finding.FixedVersion
			}
			if called[id].module == "" {
				called[id].module = firstModule(msg.Finding.Trace)
			}
		}
	}
	for id, info := range called {
		info.summary = summaries[id]
	}
	return called, nil
}

func parseAllFindings(r io.Reader) (map[string]*vulnInfo, error) {
	dec := json.NewDecoder(r)
	summaries := make(map[string]string)
	found := make(map[string]*vulnInfo)
	for {
		var msg Message
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if msg.OSV != nil && msg.OSV.ID != "" {
			summaries[msg.OSV.ID] = msg.OSV.Summary
		}
		if msg.Finding == nil || msg.Finding.OSV == "" {
			continue
		}
		id := msg.Finding.OSV
		if found[id] == nil {
			found[id] = &vulnInfo{id: id}
		}
		if semver.Compare(msg.Finding.FixedVersion, found[id].fixedVersion) > 0 {
			found[id].fixedVersion = msg.Finding.FixedVersion
		}
		if found[id].module == "" {
			found[id].module = firstModule(msg.Finding.Trace)
		}
	}
	for id, info := range found {
		info.summary = summaries[id]
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

func getBackendBinaries(pass *analysis.Pass) ([]string, error) {
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

func reportSourceFindings(pass *analysis.Pass, findings map[string]*vulnInfo) {
	if len(findings) == 0 {
		return
	}
	modGroups, stdlibGroup := splitGroups(groupByDep(findings))
	var lines []string
	if stdlibGroup != nil {
		lines = append(lines, "Update Go toolchain to "+goToolchainVersion(stdlibGroup.fixedVersion)+" or later ("+strings.Join(stdlibGroup.ids, ", ")+")")
	}
	if len(modGroups) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Update the following dependencies:")
		for _, g := range modGroups {
			lines = append(lines, "• "+depVersion(g)+" ("+strings.Join(g.ids, ", ")+")")
		}
	}
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, "Run `govulncheck ./...` in your plugin source for full details.")
	pass.ReportResult(
		pass.AnalyzerName,
		govulncheckIssueFound,
		fmt.Sprintf("govulncheck source scan reports %d reachable vulnerabilit%s", len(findings), pluralY(len(findings))),
		strings.Join(lines, "\n"),
	)
}

func reportBinaryFindings(pass *analysis.Pass, findings map[string]*vulnInfo) {
	if len(findings) == 0 {
		return
	}
	modGroups, stdlibGroup := splitGroups(groupByDep(findings))
	var lines []string
	if stdlibGroup != nil {
		lines = append(lines, "Update Go toolchain to "+goToolchainVersion(stdlibGroup.fixedVersion)+" or later ("+strings.Join(stdlibGroup.ids, ", ")+")")
	}
	if len(modGroups) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Update the following dependencies:")
		for _, g := range modGroups {
			lines = append(lines, "• "+depVersion(g)+" ("+strings.Join(g.ids, ", ")+")")
		}
	}
	pass.ReportResult(
		pass.AnalyzerName,
		govulncheckIssueFound,
		fmt.Sprintf("govulncheck binary scan reports %d vulnerabilit%s", len(findings), pluralY(len(findings))),
		strings.Join(lines, "\n"),
	)
}

type depGroup struct {
	module       string
	fixedVersion string
	ids          []string
}

// groupByDep groups vulns by their vulnerable module, taking the maximum fix
// version per module so the user sees one upgrade target per dependency.
func groupByDep(findings map[string]*vulnInfo) []depGroup {
	byModule := make(map[string]*depGroup)
	for _, info := range findings {
		mod := info.module
		if mod == "" || mod == "std" {
			mod = "stdlib"
		}
		g, ok := byModule[mod]
		if !ok {
			byModule[mod] = &depGroup{module: mod, fixedVersion: info.fixedVersion, ids: []string{info.id}}
			continue
		}
		if semver.Compare(info.fixedVersion, g.fixedVersion) > 0 {
			g.fixedVersion = info.fixedVersion
		}
		g.ids = append(g.ids, info.id)
	}
	groups := make([]depGroup, 0, len(byModule))
	for _, g := range byModule {
		sort.Strings(g.ids)
		groups = append(groups, *g)
	}
	// Non-stdlib modules alphabetically first, stdlib last.
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].module == "stdlib" {
			return false
		}
		if groups[j].module == "stdlib" {
			return true
		}
		return groups[i].module < groups[j].module
	})
	return groups
}

// goToolchainVersion strips the "v" prefix from a Go stdlib fix version for
// go.mod-compatible display (e.g. "v1.26.4" → "1.26.4"). Returns
// "unknown" when no fixed version is available so the output is never blank.
func goToolchainVersion(fixedVersion string) string {
	v := strings.TrimPrefix(fixedVersion, "v")
	if v == "" {
		return "unknown"
	}
	return v
}

// depVersion formats a module dep group for display, e.g. "golang.org/x/net v0.55.0".
// Falls back to "<module> (no fixed version)" when fixedVersion is absent.
func depVersion(g depGroup) string {
	if g.fixedVersion == "" {
		return g.module + " (no fixed version available)"
	}
	return g.module + " " + g.fixedVersion
}

// splitGroups separates module dep groups from the stdlib group.
func splitGroups(groups []depGroup) (modGroups []depGroup, stdlib *depGroup) {
	for i := range groups {
		if groups[i].module == "stdlib" {
			stdlib = &groups[i]
		} else {
			modGroups = append(modGroups, groups[i])
		}
	}
	return
}

func firstModule(trace []Frame) string {
	for _, f := range trace {
		if f.Module != "" {
			return f.Module
		}
	}
	return ""
}

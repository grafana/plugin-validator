package reactcompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/logme"
)

// reactDetectVersion is the pinned version of @grafana/react-detect.
// Bump intentionally when adopting new detection rules.
const reactDetectVersion = "0.6.4"

var (
	react19Issue = &analysis.Rule{
		Name:     "react-19-issue",
		Severity: analysis.Warning,
	}
	react19Compatible = &analysis.Rule{
		Name:     "react-19-compatible",
		Severity: analysis.OK,
	}
)

// Analyzer checks for React 19 compatibility issues in the plugin bundle by
// delegating to npx @grafana/react-detect. Silently skips if npx is not in PATH.
// If react-detect is found but fails, a warning diagnostic is emitted.
var Analyzer = &analysis.Analyzer{
	Name:     "reactcompat",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{react19Issue, react19Compatible},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "React 19 Compatibility",
		Description:  "Detects usage of React APIs removed or deprecated in React 19 using @grafana/react-detect.",
		Dependencies: "[npx](https://docs.npmjs.com/cli/v10/commands/npx)",
	},
}

// reactDetectOutput is the top-level JSON structure emitted by @grafana/react-detect.
type reactDetectOutput struct {
	SourceCodeIssues map[string][]sourceCodeIssue `json:"sourceCodeIssues"`
	DependencyIssues []dependencyIssue            `json:"dependencyIssues"`
}

type sourceCodeIssue struct {
	Pattern  string   `json:"pattern"`
	Severity string   `json:"severity"`
	Location location `json:"location"`
	Problem  string   `json:"problem"`
	Fix      fix      `json:"fix"`
	Link     string   `json:"link"`
}

type location struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type fix struct {
	Description string `json:"description"`
}

type dependencyIssue struct {
	Pattern      string   `json:"pattern"`
	Severity     string   `json:"severity"`
	Problem      string   `json:"problem"`
	Link         string   `json:"link"`
	PackageNames []string `json:"packageNames"`
}

func run(pass *analysis.Pass) (any, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok || archiveDir == "" {
		return nil, nil
	}

	npxPath, err := exec.LookPath("npx")
	if err != nil {
		// npx not in PATH is expected in environments without Node.js (e.g. Docker builder).
		// Only log at debug level — not a failure, just an unavailable optional check.
		logme.DebugFln("npx not found in PATH, skipping react-detect")
		return nil, nil
	}
	logme.DebugFln("npx path: %s", npxPath)

	output, err := runReactDetect(npxPath, archiveDir)
	if err != nil {
		logme.DebugFln("react-detect failed: %v", err)
		// Missing source maps is not a tool failure — it just means there's
		// nothing to analyze (e.g. unbuilt plugin or empty archive). Skip silently.
		if strings.Contains(err.Error(), "No source map files found") {
			return nil, nil
		}
		pass.ReportResult(
			pass.AnalyzerName,
			react19Issue,
			"React 19 compatibility: skipped (react-detect failed)",
			fmt.Sprintf("react-detect could not be executed: %v", err),
		)
		return nil, nil
	}

	issueCount := reportIssues(pass, output, archiveDir)

	if issueCount == 0 && react19Compatible.ReportAll {
		pass.ReportResult(
			pass.AnalyzerName,
			react19Compatible,
			"Plugin is compatible with React 19",
			"No React 19 compatibility issues were detected.",
		)
	}

	return nil, nil
}

// runReactDetect shells out to react-detect and returns the parsed output.
// The command's cwd is set to archiveDir so that react-detect resolves
// source-map-relative paths against the archive (yielding paths like
// <archiveDir>/src/...) rather than against the caller's cwd. The archive
// prefix is stripped later in reportIssues for reproducible output.
func runReactDetect(npxPath, archiveDir string) (*reactDetectOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// --json: machine-readable output. --skipBuildTooling: avoid running bundlers.
	// --noErrorExitCode: always exit 0 so we can parse partial output on warnings.
	// --distDir: "." because we set cmd.Dir = archiveDir below.
	// Dependency issues are intentionally included (no --skipDependencies).
	args := []string{
		"-y",
		"@grafana/react-detect@" + reactDetectVersion,
		"--json",
		"--distDir", ".",
		"--skipBuildTooling",
		"--noErrorExitCode",
	}
	logme.DebugFln("running react-detect with args: %v", args)

	cmd := exec.CommandContext(ctx, npxPath, args...)
	cmd.Dir = archiveDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if stderr.Len() > 0 {
		logme.DebugFln("react-detect stderr: %s", stderr.String())
	}
	if err != nil {
		// react-detect writes user-facing errors to stdout, not stderr, so include
		// both streams in the wrapped error for downstream detection.
		return nil, fmt.Errorf("react-detect exited with error: %w (stdout: %s) (stderr: %s)", err, string(out), stderr.String())
	}

	return parseResults(out)
}

// parseResults unmarshals the raw JSON bytes from react-detect.
func parseResults(data []byte) (*reactDetectOutput, error) {
	var output reactDetectOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parse react-detect output: %w", err)
	}
	return &output, nil
}

// reportIssues translates the react-detect output into pass diagnostics and
// returns the total number of issues reported. archiveDir is stripped from
// reported file paths so output is reproducible across machines.
func reportIssues(pass *analysis.Pass, output *reactDetectOutput, archiveDir string) int {
	// react19Issue serves as the config gate for all dynamic react-19 rules.
	if react19Issue.Disabled {
		return 0
	}

	if output == nil {
		return 0
	}

	count := 0

	patterns := make([]string, 0, len(output.SourceCodeIssues))
	for p := range output.SourceCodeIssues {
		patterns = append(patterns, p)
	}
	slices.Sort(patterns)

	for _, pattern := range patterns {
		for _, issue := range output.SourceCodeIssues[pattern] {
			rule := &analysis.Rule{
				Name:     fmt.Sprintf("react-19-%s", issue.Pattern),
				Severity: react19Issue.Severity,
			}
			detail := fmt.Sprintf(
				"Detected in %s at line %d. %s See: %s Note: this may be a false positive.",
				relativeToArchive(issue.Location.File, archiveDir),
				issue.Location.Line,
				issue.Fix.Description,
				issue.Link,
			)
			pass.ReportResult(pass.AnalyzerName, rule, "React 19 compatibility: "+issue.Problem, detail)
			count++
		}
	}

	for _, issue := range output.DependencyIssues {
		rule := &analysis.Rule{
			Name:     fmt.Sprintf("react-19-dep-%s", issue.Pattern),
			Severity: react19Issue.Severity,
		}
		detail := fmt.Sprintf(
			"Affected packages: %s. See: %s Note: this may be a false positive.",
			strings.Join(issue.PackageNames, ", "),
			issue.Link,
		)
		pass.ReportResult(pass.AnalyzerName, rule, "React 19 compatibility: "+issue.Problem, detail)
		count++
	}

	return count
}

// relativeToArchive strips the archive directory prefix from a file path
// emitted by react-detect, so reported paths are reproducible across machines.
// Falls back to the original path if it doesn't share the archive prefix.
func relativeToArchive(file, archiveDir string) string {
	if archiveDir == "" {
		return file
	}
	prefix := strings.TrimRight(archiveDir, "/") + "/"
	if strings.HasPrefix(file, prefix) {
		return strings.TrimPrefix(file, prefix)
	}
	return file
}

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
// delegating to npx @grafana/react-detect. If npx is not available or
// react-detect fails, a warning diagnostic is emitted explaining the skip.
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
		logme.DebugFln("npx not found in PATH, skipping react-detect")
		pass.ReportResult(
			pass.AnalyzerName,
			react19Issue,
			"React 19 compatibility: skipped (npx not found in PATH)",
			"Install Node.js and npx to enable React 19 compatibility checks.",
		)
		return nil, nil
	}
	logme.DebugFln("npx path: %s", npxPath)

	output, err := runReactDetect(npxPath, archiveDir)
	if err != nil {
		logme.DebugFln("react-detect failed: %v", err)
		pass.ReportResult(
			pass.AnalyzerName,
			react19Issue,
			"React 19 compatibility: skipped (react-detect failed)",
			fmt.Sprintf("react-detect could not be executed: %v", err),
		)
		return nil, nil
	}

	issueCount := reportIssues(pass, output)

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
// --distDir points react-detect directly at the extracted archive directory,
// avoiding the need for a symlink or temp directory.
func runReactDetect(npxPath, archiveDir string) (*reactDetectOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// --json: machine-readable output. --skipBuildTooling: avoid running bundlers.
	// --noErrorExitCode: always exit 0 so we can parse partial output on warnings.
	// --distDir: point at the extracted archive directly (available since v0.6.4).
	// Dependency issues are intentionally included (no --skipDependencies).
	args := []string{
		"-y",
		// Pinned for reproducibility. Bump intentionally when adopting new detection rules.
		"@grafana/react-detect@0.6.4",
		"--json",
		"--distDir", archiveDir,
		"--skipBuildTooling",
		"--noErrorExitCode",
	}
	logme.DebugFln("running react-detect with args: %v", args)

	cmd := exec.CommandContext(ctx, npxPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if stderr.Len() > 0 {
		logme.DebugFln("react-detect stderr: %s", stderr.String())
	}
	if err != nil {
		return nil, fmt.Errorf("react-detect exited with error: %w (stderr: %s)", err, stderr.String())
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
// returns the total number of issues reported.
func reportIssues(pass *analysis.Pass, output *reactDetectOutput) int {
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
				issue.Location.File,
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

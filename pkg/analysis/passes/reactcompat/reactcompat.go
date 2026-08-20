package reactcompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/logme"
)

const (
	// reactDetectVersion is the pinned version of @grafana/react-detect.
	// Bump intentionally when adopting new detection rules.
	reactDetectVersion = "0.6.4"
	// Limit each react-detect result so downstream payloads remain bounded.
	maxReportBytes                = 32 * 1024
	maxSummaryDiagnosticBytes     = 1024
	maxCombinedStreamPreviewBytes = 30 * 1024
	maxDebugStreamPreviewBytes    = 8 * 1024
	maxStdoutCaptureBytes         = 16 * 1024 * 1024
	maxStderrCaptureBytes         = 1024 * 1024
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

// cappedBuffer accepts all writes but retains only the configured byte limit.
type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *cappedBuffer) Bytes() []byte {
	return b.buf.Bytes()
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
		reportExecutionFailure(pass, err)
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
	stdout := &cappedBuffer{limit: maxStdoutCaptureBytes}
	stderr := &cappedBuffer{limit: maxStderrCaptureBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	out := stdout.Bytes()
	if len(stderr.Bytes()) > 0 {
		logme.DebugFln("react-detect stderr: %s", streamPreview(stderr.Bytes(), maxDebugStreamPreviewBytes))
	}
	if stdout.truncated {
		captureErr := fmt.Errorf("stdout exceeded the %d byte capture limit", maxStdoutCaptureBytes)
		if err != nil {
			captureErr = fmt.Errorf("%w; %v", err, captureErr)
		}
		return nil, formatExecutionError(captureErr, out, stderr.Bytes())
	}
	if err != nil {
		output, parseErr := parseResults(out)
		if parseErr == nil {
			logme.DebugFln(
				"react-detect exited with error after writing valid JSON: %v (stdout: %d bytes)",
				err,
				len(out),
			)
			return output, nil
		}
		return nil, formatExecutionError(err, out, stderr.Bytes())
	}

	return parseResults(out)
}

func formatExecutionError(runErr error, stdout, stderr []byte) error {
	stdoutLimit, stderrLimit := streamPreviewLimits(len(stdout), len(stderr))
	return fmt.Errorf(
		"react-detect exited with error: %w (stdout: %s) (stderr: %s)",
		runErr,
		streamPreview(stdout, stdoutLimit),
		streamPreview(stderr, stderrLimit),
	)
}

func streamPreviewLimits(stdoutBytes, stderrBytes int) (int, int) {
	half := maxCombinedStreamPreviewBytes / 2
	stdoutLimit := min(stdoutBytes, half)
	stderrLimit := min(stderrBytes, half)
	remaining := maxCombinedStreamPreviewBytes - stdoutLimit - stderrLimit

	stdoutExtra := min(stdoutBytes-stdoutLimit, remaining)
	stdoutLimit += stdoutExtra
	remaining -= stdoutExtra
	stderrLimit += min(stderrBytes-stderrLimit, remaining)
	return stdoutLimit, stderrLimit
}

func streamPreview(data []byte, limit int) string {
	if len(data) == 0 {
		return "empty"
	}
	if len(data) <= limit {
		return fmt.Sprintf("%d bytes: %s", len(data), data)
	}
	return fmt.Sprintf("%d bytes: %s...[truncated]", len(data), data[:limit])
}

// parseResults unmarshals the raw JSON bytes from react-detect.
func parseResults(data []byte) (*reactDetectOutput, error) {
	var fields struct {
		SourceCodeIssues json.RawMessage `json:"sourceCodeIssues"`
		DependencyIssues json.RawMessage `json:"dependencyIssues"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("parse react-detect output: %w", err)
	}
	if len(fields.SourceCodeIssues) == 0 || len(fields.DependencyIssues) == 0 ||
		bytes.Equal(bytes.TrimSpace(fields.SourceCodeIssues), []byte("null")) ||
		bytes.Equal(bytes.TrimSpace(fields.DependencyIssues), []byte("null")) {
		return nil, fmt.Errorf("parse react-detect output: missing required result fields")
	}

	var output reactDetectOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parse react-detect output: %w", err)
	}
	return &output, nil
}

func reportExecutionFailure(pass *analysis.Pass, runErr error) {
	diagnostic := analysis.Diagnostic{
		Name:     react19Issue.Name,
		Severity: react19Issue.Severity,
		Title:    "React 19 compatibility: skipped (react-detect failed)",
		Detail:   fmt.Sprintf("react-detect could not be executed: %v", runErr),
	}
	diagnostic = limitDiagnosticSize(diagnostic, maxReportBytes-2)
	reportDiagnostic(pass, diagnostic)
}

func reportDiagnostic(pass *analysis.Pass, diagnostic analysis.Diagnostic) {
	if react19Issue.Disabled {
		return
	}
	pass.ReportResult(
		pass.AnalyzerName,
		&analysis.Rule{Name: diagnostic.Name, Severity: diagnostic.Severity},
		diagnostic.Title,
		diagnostic.Detail,
	)
}

func diagnosticJSONSize(diagnostic analysis.Diagnostic) int {
	data, err := json.Marshal(diagnostic)
	if err != nil {
		return maxReportBytes + 1
	}
	return len(data)
}

func diagnosticsJSONSize(diagnostics []analysis.Diagnostic) int {
	data, err := json.Marshal(diagnostics)
	if err != nil {
		return maxReportBytes + 1
	}
	return len(data)
}

func limitDiagnosticSize(diagnostic analysis.Diagnostic, maxBytes int) analysis.Diagnostic {
	if diagnosticJSONSize(diagnostic) <= maxBytes {
		return diagnostic
	}

	const marker = "\n...[truncated]"
	detail := []rune(diagnostic.Detail)
	low, high := 0, len(detail)
	for low < high {
		mid := (low + high + 1) / 2
		candidate := diagnostic
		candidate.Detail = string(detail[:mid]) + marker
		if diagnosticJSONSize(candidate) <= maxBytes {
			low = mid
		} else {
			high = mid - 1
		}
	}
	diagnostic.Detail = string(detail[:low]) + marker
	return diagnostic
}

// reportIssues translates the react-detect output into pass diagnostics and
// returns the total number of detected issues before the report size limit.
// archiveDir is stripped from reported file paths so output is reproducible.
func reportIssues(pass *analysis.Pass, output *reactDetectOutput, archiveDir string) int {
	// react19Issue serves as the config gate for all dynamic react-19 rules.
	if react19Issue.Disabled || output == nil {
		return 0
	}

	diagnostics := issueDiagnostics(output, archiveDir)
	total := len(diagnostics)
	if diagnosticsJSONSize(diagnostics) > maxReportBytes {
		diagnostics = limitIssueDiagnostics(diagnostics)
	}
	for _, diagnostic := range diagnostics {
		reportDiagnostic(pass, diagnostic)
	}
	return total
}

func limitIssueDiagnostics(diagnostics []analysis.Diagnostic) []analysis.Diagnostic {
	const arrayJSONBytes = 2
	issueBudget := maxReportBytes - maxSummaryDiagnosticBytes - arrayJSONBytes - 1
	used := 0
	reported := 0
	for _, diagnostic := range diagnostics {
		size := diagnosticJSONSize(diagnostic)
		if reported > 0 {
			size++
		}
		if used+size > issueBudget {
			break
		}
		used += size
		reported++
	}

	omitted := len(diagnostics) - reported
	summary := limitDiagnosticSize(analysis.Diagnostic{
		Name:     react19Issue.Name,
		Severity: react19Issue.Severity,
		Title:    "React 19 compatibility: results truncated",
		Detail: fmt.Sprintf(
			"Reported %d of %d issues. %d issues were omitted because react-detect reports are limited to %d bytes.",
			reported,
			len(diagnostics),
			omitted,
			maxReportBytes,
		),
	}, maxSummaryDiagnosticBytes)

	limited := make([]analysis.Diagnostic, 0, reported+1)
	limited = append(limited, diagnostics[:reported]...)
	return append(limited, summary)
}

func issueDiagnostics(output *reactDetectOutput, archiveDir string) []analysis.Diagnostic {
	if output == nil {
		return nil
	}

	var diagnostics []analysis.Diagnostic
	patterns := make([]string, 0, len(output.SourceCodeIssues))
	for pattern := range output.SourceCodeIssues {
		patterns = append(patterns, pattern)
	}
	slices.Sort(patterns)

	for _, pattern := range patterns {
		for _, issue := range output.SourceCodeIssues[pattern] {
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Name:     fmt.Sprintf("react-19-%s", issue.Pattern),
				Severity: react19Issue.Severity,
				Title:    "React 19 compatibility: " + issue.Problem,
				Detail: fmt.Sprintf(
					"Detected in %s at line %d. %s See: %s Note: this may be a false positive.",
					relativeToArchive(issue.Location.File, archiveDir),
					issue.Location.Line,
					issue.Fix.Description,
					issue.Link,
				),
			})
		}
	}

	for _, issue := range output.DependencyIssues {
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Name:     fmt.Sprintf("react-19-dep-%s", issue.Pattern),
			Severity: react19Issue.Severity,
			Title:    "React 19 compatibility: " + issue.Problem,
			Detail: fmt.Sprintf(
				"Affected packages: %s. See: %s Note: this may be a false positive.",
				strings.Join(issue.PackageNames, ", "),
				issue.Link,
			),
		})
	}

	return diagnostics
}

// relativeToArchive strips the archive directory prefix from a file path
// emitted by react-detect, so reported paths are reproducible across machines.
// react-detect may report paths with symlinks resolved (on macOS the temp dir
// /var/folders/... resolves to /private/var/folders/...), so the resolved
// archiveDir is tried as well. Falls back to the original path if it doesn't
// share the archive prefix.
func relativeToArchive(file, archiveDir string) string {
	if archiveDir == "" {
		return file
	}
	dirs := []string{archiveDir}
	if resolved, err := filepath.EvalSymlinks(archiveDir); err == nil && resolved != archiveDir {
		dirs = append(dirs, resolved)
	}
	for _, dir := range dirs {
		prefix := strings.TrimRight(dir, "/") + "/"
		if rel, ok := strings.CutPrefix(file, prefix); ok {
			return rel
		}
	}
	return file
}

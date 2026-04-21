package plugindocs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

const (
	cliPackage    = "@grafana/plugin-docs-cli"
	cliRunCommand = "validate"
	// runTimeout bounds the full `npx --yes ...` invocation, including package download on first run.
	runTimeout = 120 * time.Second
)

var (
	pluginDocsError = &analysis.Rule{
		Name:     "plugin-docs-error",
		Severity: analysis.Error,
	}
	pluginDocsWarning = &analysis.Rule{
		Name:     "plugin-docs-warning",
		Severity: analysis.Warning,
	}
	pluginDocsInfo = &analysis.Rule{
		Name:     "plugin-docs-info",
		Severity: analysis.Recommendation,
	}
	pluginDocsCliFailure = &analysis.Rule{
		Name:     "plugin-docs-cli-failure",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "plugindocs",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		pluginDocsError,
		pluginDocsWarning,
		pluginDocsInfo,
		pluginDocsCliFailure,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Plugin Docs",
		Description:  fmt.Sprintf("Runs the `%s validate` command to check multi-page documentation (only for plugins that set `docsPath` in `plugin.json`).", cliPackage),
		Dependencies: "`node`, `npx`",
	},
}

// cliDiagnostic mirrors the Diagnostic shape produced by `plugin-docs-cli validate --json`.
// See plugin-tools/packages/plugin-docs-cli/src/validation/types.ts.
type cliDiagnostic struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

type cliResult struct {
	Valid       bool            `json:"valid"`
	Diagnostics []cliDiagnostic `json:"diagnostics"`
}

func run(pass *analysis.Pass) (interface{}, error) {
	if os.Getenv("SKIP_PLUGIN_DOCS_CLI") != "" {
		logme.Debugln("SKIP_PLUGIN_DOCS_CLI set, skipping plugin docs validation")
		return nil, nil
	}

	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok || sourceCodeDir == "" {
		// no source code available - can't validate docs
		return nil, nil
	}

	// hard gate: only run if the plugin has opted in via `docsPath` in src/plugin.json.
	// this short-circuits before any external process is spawned for the ~99% of plugins
	// that haven't opted into multi-page docs.
	hasDocs, err := pluginHasDocsPath(sourceCodeDir)
	if err != nil {
		logme.Debugln("plugindocs: failed to inspect src/plugin.json, skipping:", err)
		return nil, nil
	}
	if !hasDocs {
		logme.Debugln("plugindocs: docsPath not set in src/plugin.json, skipping")
		return nil, nil
	}

	npxBin, err := exec.LookPath("npx")
	if err != nil {
		logme.Debugln("plugindocs: npx not found on PATH, skipping plugin docs validation")
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, npxBin, "--yes", cliPackage, cliRunCommand, "--json", "--strict")
	cmd.Dir = sourceCodeDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// the CLI exits 1 when any `error` severity diagnostic is present, or when something
	// goes wrong before validation (e.g. could not find src/plugin.json). distinguish by
	// whether stdout contains parseable JSON.
	var result cliResult
	if jsonErr := json.Unmarshal(stdout.Bytes(), &result); jsonErr != nil {
		reportCliFailure(pass, runErr, stderr.String(), stdout.String())
		return nil, nil
	}

	for _, d := range result.Diagnostics {
		rule := ruleForSeverity(d.Severity)
		pass.ReportResult(
			pass.AnalyzerName,
			rule,
			formatTitle(d),
			d.Detail,
		)
	}

	return nil, nil
}

// pluginHasDocsPath reports whether src/plugin.json exists and has a non-empty docsPath field.
// returns (false, nil) when plugin.json is missing - a missing src/plugin.json means the
// validator was invoked without source code, which is a skip condition, not an error.
func pluginHasDocsPath(sourceCodeDir string) (bool, error) {
	pluginJSONPath := filepath.Join(sourceCodeDir, "src", "plugin.json")
	raw, err := os.ReadFile(pluginJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read %s: %w", pluginJSONPath, err)
	}

	var parsed struct {
		DocsPath string `json:"docsPath"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false, fmt.Errorf("parse %s: %w", pluginJSONPath, err)
	}

	return strings.TrimSpace(parsed.DocsPath) != "", nil
}

func ruleForSeverity(severity string) *analysis.Rule {
	switch severity {
	case "error":
		return pluginDocsError
	case "warning":
		return pluginDocsWarning
	case "info":
		return pluginDocsInfo
	default:
		// unknown severity from the CLI - surface it as a warning so it's visible
		// without blocking publishing. log for debugging.
		logme.DebugFln("plugindocs: unknown CLI severity %q, defaulting to warning", severity)
		return pluginDocsWarning
	}
}

// formatTitle composes a human-readable title that preserves the CLI rule name and
// file:line origin, since a single generic validator rule wraps many CLI rules.
func formatTitle(d cliDiagnostic) string {
	var location string
	if d.File != "" {
		if d.Line > 0 {
			location = fmt.Sprintf(" (%s:%d)", d.File, d.Line)
		} else {
			location = fmt.Sprintf(" (%s)", d.File)
		}
	}
	return fmt.Sprintf("[%s] %s%s", d.Rule, d.Title, location)
}

func reportCliFailure(pass *analysis.Pass, runErr error, stderr, stdout string) {
	var msg strings.Builder
	msg.WriteString("Could not run `")
	msg.WriteString(cliPackage)
	msg.WriteString(" ")
	msg.WriteString(cliRunCommand)
	msg.WriteString("`.")
	if runErr != nil {
		msg.WriteString(" error: ")
		msg.WriteString(runErr.Error())
	}
	detail := strings.TrimSpace(stderr)
	if detail == "" {
		detail = strings.TrimSpace(stdout)
	}
	if detail == "" {
		detail = "No output captured from the CLI."
	}
	// bound the detail so a noisy CLI failure doesn't produce a multi-MB diagnostic.
	const maxDetail = 4096
	if len(detail) > maxDetail {
		detail = detail[:maxDetail] + "...[truncated]"
	}

	pass.ReportResult(
		pass.AnalyzerName,
		pluginDocsCliFailure,
		msg.String(),
		detail,
	)
}

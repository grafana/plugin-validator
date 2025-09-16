package coderules

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

//go:embed semgrep-rules.yaml
var semgrepRules string

var (
	codeRulesViolationError = &analysis.Rule{
		Name:     "code-rules-violation-error",
		Severity: analysis.Error,
	}
	codeRulesViolationWarning = &analysis.Rule{
		Name:     "code-rules-violation-warning",
		Severity: analysis.Warning,
	}
	noCodeRulesViolations = &analysis.Rule{
		Name:     "no-code-rules-violations",
		Severity: analysis.OK,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "code-rules",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		codeRulesViolationError,
		codeRulesViolationWarning,
		noCodeRulesViolations,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Code Rules",
		Description:  "Checks for forbidden access to environment variables, file system or use of syscall module.",
		Dependencies: "[semgrep](https://github.com/returntocorp/semgrep), `sourceCodeUri`",
	},
}

func run(pass *analysis.Pass) (any, error) {
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		// no source code for the validator
		return nil, nil
	}

	// check if there's a pkg folder inside the source code
	sourceCodePkgDir := filepath.Clean(fmt.Sprintf("%s/pkg/", sourceCodeDir))
	info, err := os.Stat(sourceCodePkgDir)
	// if there's a pkg folder, use it as the source code dir
	if err == nil && info.IsDir() {
		sourceCodeDir = sourceCodePkgDir
	}

	logme.DebugFln("source code dir: %s", sourceCodeDir)

	semgrepPath, err := getSemgrepPath()
	if err != nil {
		return nil, nil
	}

	logme.DebugFln("semgrep path: %s", semgrepPath)

	semgrepRulesPath, cleanup, err := getSemgrepRulesPath()
	if err != nil {
		return nil, nil
	}
	if cleanup != nil {
		defer cleanup()
	}
	logme.DebugFln("semgrep rules path: %s", semgrepRulesPath)

	// run semgrep against the source code
	semGrepArgs := []string{
		"--json",
		"--quiet",
		"--metrics",
		"off",
		"--config",
		semgrepRulesPath,
		sourceCodeDir,
	}
	logme.DebugFln("semgrep args: %v", semGrepArgs)
	cmd := exec.Command(semgrepPath, semGrepArgs...)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	// unmarshal semgrep output
	var semgrepResults SemgrepResults
	err = json.Unmarshal(out, &semgrepResults)
	if err != nil {
		return nil, nil
	}

	violations := 0

	// report semgrep results
	for _, result := range semgrepResults.Results {

		severity := strings.ToLower(result.Extra.Severity)
		switch severity {
		case "error":
			pass.ReportResult(
				pass.AnalyzerName,
				codeRulesViolationError,
				result.Extra.Message,
				fmt.Sprintf(
					"Code rule violation found in %s at line %d",
					result.Path,
					result.Start.Line,
				),
			)
			violations++
		case "warning":
			pass.ReportResult(
				pass.AnalyzerName,
				codeRulesViolationWarning,
				result.Extra.Message,
				fmt.Sprintf(
					"Code rule violation found in %s at line %d",
					result.Path,
					result.Start.Line,
				),
			)
			violations++
		default:
			pass.ReportResult(
				pass.AnalyzerName,
				codeRulesViolationWarning,
				result.Extra.Message,
				fmt.Sprintf(
					"Code rule violation found in %s at line %d",
					result.Path,
					result.Start.Line,
				),
			)
		}
	}

	if violations == 0 && noCodeRulesViolations.ReportAll {
		noCodeRulesViolations.Severity = analysis.OK
		pass.ReportResult(
			pass.AnalyzerName,
			noCodeRulesViolations,
			"no code rules violations found",
			"semgrep didn't find any code rules violations",
		)
	}

	// no need to return anything
	return nil, nil

}

func getSemgrepPath() (string, error) {
	path, err := exec.LookPath("semgrep")
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("semgrep not found in PATH")
	}
	return path, nil
}

func getSemgrepRulesPath() (string, func(), error) {
	if semgrepRules == "" {
		return "", nil, fmt.Errorf("semgrep rules not found")
	}
	// write semgrep rules to a temp file
	tmpFile, err := os.CreateTemp(os.TempDir(), "*semgrep-rules.yaml")
	if err != nil {
		return "", nil, err
	}
	_, err = tmpFile.WriteString(semgrepRules)
	if err != nil {
		return "", nil, err
	}
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil

}

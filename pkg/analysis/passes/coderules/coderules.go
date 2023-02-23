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
	codeRulesViolation            = &analysis.Rule{Name: "code-rules-violation", Severity: analysis.Error}
	codeRulesViolationAccesingEnv = &analysis.Rule{Name: "code-rules-violation-env", Severity: analysis.Warning}
	semgrepNotFound               = &analysis.Rule{Name: "semgrep-not-found", Severity: analysis.Warning}
	semgrepRunningErr             = &analysis.Rule{Name: "semgrep-running-err", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "code-rules",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{codeRulesViolation},
}

func run(pass *analysis.Pass) (interface{}, error) {
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
		// can't run semgrep
		if semgrepNotFound.ReportAll {
			pass.ReportResult(pass.AnalyzerName, semgrepNotFound, "semgrep not found in PATH", "")
		}
		return nil, nil
	}

	logme.DebugFln("semgrep path: %s", semgrepPath)

	semgrepRulesPath, cleanup, err := getSemgrepRulesPath()
	if err != nil {
		// can't run semgrep
		if semgrepNotFound.ReportAll {
			pass.ReportResult(pass.AnalyzerName, semgrepNotFound, "semgrep rules not found. Bad binary compilation?", "")
		}
		return nil, nil
	}
	if cleanup != nil {
		defer cleanup()
	}
	logme.DebugFln("semgrep rules path: %s", semgrepRulesPath)

	// run semgrep against the source code
	semGrepArgs := []string{
		"--json",
		"--lang",
		"go",
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
		// semgrep failed to run
		if semgrepRunningErr.ReportAll {
			pass.ReportResult(pass.AnalyzerName, semgrepRunningErr, "semgrep failed to run", err.Error())
		}
		return nil, nil
	}
	// unmarshal semgrep output
	var semgrepResults SemgrepResults
	err = json.Unmarshal(out, &semgrepResults)
	if err != nil {
		// semgrep output is not valid json
		if semgrepRunningErr.ReportAll {
			pass.ReportResult(pass.AnalyzerName, semgrepRunningErr, "semgrep output is not valid json", err.Error())
		}
		return nil, nil
	}

	// report semgrep results
	for _, result := range semgrepResults.Results {

		ruleIdSplit := strings.Split(result.Check_id, ".")
		ruleName := ""
		if len(ruleIdSplit) == 2 {
			ruleName = ruleIdSplit[1]
		}

		switch ruleName {
		case "access-only-allowed-os-environment":
			pass.ReportResult(pass.AnalyzerName, codeRulesViolationAccesingEnv, result.Extra.Message, fmt.Sprintf("Code rule violation found in %s at line %d", result.Path, result.Start.Line))
		default:
			pass.ReportResult(pass.AnalyzerName, codeRulesViolation, result.Extra.Message, fmt.Sprintf("Code rule violation found in %s at line %d", result.Path, result.Start.Line))
		}

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

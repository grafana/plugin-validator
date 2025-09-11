package gosec

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	goSecNotInstalled   = &analysis.Rule{Name: "go-sec-not-installed", Severity: analysis.Warning}
	goSectIssueFound    = &analysis.Rule{Name: "go-sec-issue-found", Severity: analysis.Warning}
	goSectNoIssuesFound = &analysis.Rule{Name: "go-sec-no-issues-found", Severity: analysis.OK}
)

// could be low, medium, high (see gosec docs)
var targetSeverity = "HIGH"

var Analyzer = &analysis.Analyzer{
	Name:     "go-sec",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{goSecNotInstalled, goSectIssueFound, goSectNoIssuesFound},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Go Security Checker",
		Description:  "Inspects source code for security problems by scanning the Go AST.",
		Dependencies: "[gosec](https://github.com/securego/gosec), `sourceCodeUri`",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// only run if sourcecode.Analyzer succeeded
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	// check if gosec is installed
	goSecBin, err := exec.LookPath("gosec")
	if err != nil {
		if goSecNotInstalled.ReportAll {
			logme.Debugln("gosec not installed, skipping gosec analysis")
			pass.ReportResult(
				pass.AnalyzerName,
				goSecNotInstalled,
				"gosec not installed",
				"Skipping gosec analysis",
			)
		}
		return nil, nil
	}

	// run gosec
	goSecCommand := exec.Command(
		goSecBin,
		"-quiet",
		"-severity",
		targetSeverity,
		"-fmt",
		"json",
		"-r",
	)
	goSecCommand.Dir = sourceCodeDir
	goSecOutput, err := goSecCommand.Output()
	if err != nil {
		// gosec exits 1 if it finds issues. If there's an error other than an exit error, return it
		_, ok = err.(*exec.ExitError)
		if !ok {
			logme.ErrorF("Error running gosec: %v", err)
			return nil, err
		}
	}

	if len(goSecOutput) == 0 {
		logme.Debugln("gosec output is empty, skipping gosec report")
		return nil, nil
	}

	var goSectResults Result
	err = json.Unmarshal(goSecOutput, &goSectResults)
	if err != nil {
		fmt.Println("Error running gosec", err)
		logme.Errorln("Error unmarshalling gosec output", "error", err)
		// breaking the validator to notify the user that the gosec output is not as expected
		return nil, err
	}

	count := 0
	brokenRules := make([]string, 0)
	for _, issue := range goSectResults.Issues {
		if strings.ToUpper(issue.Severity) == targetSeverity {
			brokenRules = append(brokenRules, issue.RuleID)
			count++
		}
	}

	if count > 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			goSectIssueFound,
			fmt.Sprintf("gosec analysis reports %d issues with %s severity", count, targetSeverity),
			fmt.Sprintf(
				"Run gosec https://github.com/securego/gosec in your plugin code to see the issues. Found issues in rules: %s",
				strings.Join(brokenRules, ", "),
			),
		)
	} else if goSectNoIssuesFound.ReportAll {
		pass.ReportResult(
			pass.AnalyzerName,
			goSectNoIssuesFound,
			"gosec analysis reports no issues",
			"",
		)
	}

	return nil, nil
}

package codediff

import (
	"fmt"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var (
	codeDiffAnalysis = &analysis.Rule{
		Name:     "code-diff-analysis",
		Severity: analysis.SuspectedProblem,
	}
	codeDiffversions = &analysis.Rule{
		Name:     "code-diff-versions",
		Severity: analysis.SuspectedProblem,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "codediff",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{codeDiffAnalysis, codeDiffversions},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Code Diff",
		Description:  "",
		Dependencies: "Google API Key with Generative AI access",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	fmt.Printf(pass.CheckParams.SourceCodeReference)
	return nil, nil
}

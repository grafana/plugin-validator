package jargon

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var (
	readmeComment = &analysis.Rule{Name: "readme-comment", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "comment",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{readmeComment},
}

func run(pass *analysis.Pass) (interface{}, error) {
	b, ok := pass.ResultOf[readme.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	readmeContent := string(b)

	commentRegex := `<!--(.*?)-->`
	re := regexp.MustCompile(commentRegex)

	// Find all matches
	comment := re.FindString(readmeContent)

	if len(comment) > 0 {
		pass.ReportResult(pass.AnalyzerName, readmeComment, "README.md contains comment(s).", "")
	}
	if readmeComment.ReportAll {
		readmeComment.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, readmeComment, "README.md contains no comment(s).", "")
	}

	return nil, nil
}

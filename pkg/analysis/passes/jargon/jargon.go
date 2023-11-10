package jargon

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var (
	developerJargon = &analysis.Rule{Name: "developer-jargon", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "jargon",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{developerJargon},
}

func run(pass *analysis.Pass) (interface{}, error) {
	jargon := []string{
		"yarn",
		"nodejs",
	}

	readmeContent, ok := pass.ResultOf[readme.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var found []string
	for _, word := range jargon {
		if bytes.Contains(readmeContent, []byte(word)) {
			found = append(found, word)
		}
	}

	if len(found) > 0 {
		foundJargon := strings.Join(found, ", ")
		reportMessage := fmt.Sprintf("README.md contains developer jargon: (%s)", foundJargon)
		explanation := "Move any developer and contributor documentation to a separate file and link to it from the README.md. For example, CONTRIBUTING.md, DEVELOPMENT.md, etc."
		pass.ReportResult(pass.AnalyzerName, developerJargon, reportMessage, explanation)
	} else {
		if developerJargon.ReportAll {
			developerJargon.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, developerJargon, "README.md contains no developer jargon", "")
		}
	}

	return nil, nil
}

package jargon

import (
	"bytes"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var (
	developerJargon = &analysis.Rule{Name: "developer-jargon"}
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

	b := pass.ResultOf[readme.Analyzer].([]byte)

	var found []string
	for _, word := range jargon {
		if bytes.Contains(b, []byte(word)) {
			found = append(found, word)
		}
	}

	if len(found) > 0 {
		foundJargon := strings.Join(found, ", ")
		reportMessage := "README.md contains developer jargon: (%s)\n>> %s"
		extraMessage := "Move any developer and contributor documentation to a separate file and link to it from the README.md. For example, CONTRIBUTING.md, DEVELOPMENT.md, etc."
		pass.Reportf(pass.AnalyzerName, developerJargon, reportMessage, foundJargon, extraMessage)
	} else {
		if developerJargon.ReportAll {
			developerJargon.Severity = analysis.OK
			pass.Reportf(pass.AnalyzerName, developerJargon, "README.md contains no developer jargon")
		}
	}

	return nil, nil
}

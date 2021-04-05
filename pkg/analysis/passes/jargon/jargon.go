package jargon

import (
	"bytes"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var Analyzer = &analysis.Analyzer{
	Name:     "jargon",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
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
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Warning,
			Message:  "remove developer jargon for more user-friendly docs",
			Context:  "README.md",
		})
	}

	return nil, nil
}

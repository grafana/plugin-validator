package templatereadme

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var Analyzer = &analysis.Analyzer{
	Name:     "templatereadme",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	readme := pass.ResultOf[readme.Analyzer].([]byte)

	re := regexp.MustCompile("^# Grafana (Panel|Data Source|Data Source Backend) Plugin Template")

	if m := re.Find(readme); m != nil {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Warning,
			Message:  "uses README from template",
			Context:  "README.md",
		})
	}

	return nil, nil
}

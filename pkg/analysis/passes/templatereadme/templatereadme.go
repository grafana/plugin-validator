package templatereadme

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var (
	templateReadme = &analysis.Rule{Name: "template-readme"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "templatereadme",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{templateReadme},
}

func run(pass *analysis.Pass) (interface{}, error) {
	readme := pass.ResultOf[readme.Analyzer].([]byte)

	re := regexp.MustCompile("^# Grafana (Panel|Data Source|Data Source Backend) Plugin Template")

	if m := re.Find(readme); m != nil {
		pass.Reportf(pass.AnalyzerName, templateReadme, "README.md: uses README from template")
	} else {
		if templateReadme.ReportAll {
			templateReadme.Severity = analysis.OK
			pass.Reportf(pass.AnalyzerName, templateReadme, "README.md: does not use README from template")
		}
	}

	return nil, nil
}

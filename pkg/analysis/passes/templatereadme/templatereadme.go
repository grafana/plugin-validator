package templatereadme

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var (
	templateReadme = &analysis.Rule{Name: "template-readme", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "templatereadme",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{templateReadme},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Unique README.md",
		Description: "Ensures the plugin doesn't re-use the template from the `create-plugin` tool.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	readme, ok := pass.ResultOf[readme.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	re := regexp.MustCompile("(?i)Grafana (Panel|Data Source|Datasource|App|Data Source Backend) Plugin Template")

	if m := re.Find(readme); m != nil {
		pass.ReportResult(pass.AnalyzerName, templateReadme, "README.md: uses README from template", "The README.md file uses the README from the plugin template. Please update it to describe your plugin.")
	} else {
		if templateReadme.ReportAll {
			templateReadme.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, templateReadme, "README.md: does not use README from template", "")
		}
	}

	return nil, nil
}

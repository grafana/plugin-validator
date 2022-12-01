package htmlreadme

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var (
	noHTMLReadme = &analysis.Rule{Name: "no-html-readme"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "htmlreadme",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{noHTMLReadme},
}

func run(pass *analysis.Pass) (interface{}, error) {
	readme := pass.ResultOf[readme.Analyzer].([]byte)

	re := regexp.MustCompile("</[a-z]+>")

	if re.Match(readme) {
		pass.ReportResult(pass.AnalyzerName, noHTMLReadme, "README.md: html is not supported and will not render correctly", "Remove all HTML from your README.md file. It will not render correctly on the Grafana catalog.")
	} else {
		if noHTMLReadme.ReportAll {
			noHTMLReadme.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, noHTMLReadme, "README.md contains no html", "")
		}
	}

	return nil, nil
}

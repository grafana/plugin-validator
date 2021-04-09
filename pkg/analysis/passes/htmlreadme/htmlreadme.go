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
		pass.Reportf(noHTMLReadme, "README.md: html is not supported and will not render correctly")
	}

	return nil, nil
}

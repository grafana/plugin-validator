package htmlreadme

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var Analyzer = &analysis.Analyzer{
	Name:     "htmlreadme",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	readme := pass.ResultOf[readme.Analyzer].([]byte)

	re := regexp.MustCompile("</[a-z]+>")

	if re.Match(readme) {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Warning,
			Message:  "html is not supported and will not render correctly",
			Context:  "README.md",
		})
	}

	return nil, nil
}

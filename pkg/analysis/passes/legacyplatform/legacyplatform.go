package legacyplatform

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
)

var Analyzer = &analysis.Analyzer{
	Name:     "legacyplatform",
	Requires: []*analysis.Analyzer{modulejs.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	module := pass.ResultOf[modulejs.Analyzer].([]byte)

	var (
		reactExp   = regexp.MustCompile(`(@grafana/data)`)
		angularExp = regexp.MustCompile(`\s(app/plugins/sdk)`)
	)

	if angularExp.Match(module) && !reactExp.Match(module) {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Warning,
			Message:  "uses legacy plugin platform",
			Context:  "module.js",
		})
	}

	return nil, nil
}

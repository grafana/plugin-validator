package legacyplatform

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
)

var (
	legacyPlatform = &analysis.Rule{Name: "legacy-platform"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "legacyplatform",
	Requires: []*analysis.Analyzer{modulejs.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{legacyPlatform},
}

func run(pass *analysis.Pass) (interface{}, error) {
	module := pass.ResultOf[modulejs.Analyzer].([]byte)

	var (
		reactExp   = regexp.MustCompile(`(@grafana/data)`)
		angularExp = regexp.MustCompile(`\s(app/plugins/sdk)`)
	)

	if angularExp.Match(module) && !reactExp.Match(module) {
		pass.Reportf(legacyPlatform, "module.js: uses legacy plugin platform")
	}

	return nil, nil
}

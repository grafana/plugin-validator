package legacyplatform

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
)

var (
	legacyPlatform = &analysis.Rule{Name: "legacy-platform", Severity: analysis.Warning}
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
		angularExp = regexp.MustCompile(`([\s"']grafana/app/)`)
	)

	if angularExp.Match(module) && !reactExp.Match(module) {
		pass.ReportResult(pass.AnalyzerName, legacyPlatform, "module.js: uses legacy plugin platform", "The plugin uses the legacy plugin platform (angularjs). Please migrate the plugin to use the new plugins platform.")
	} else {
		if legacyPlatform.ReportAll {
			legacyPlatform.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, legacyPlatform, "module.js: uses current plugin platform", "")
		}
	}

	return nil, nil
}

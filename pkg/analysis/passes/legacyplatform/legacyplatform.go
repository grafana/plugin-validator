package legacyplatform

import (
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
)

var (
	legacyPlatform = &analysis.Rule{Name: "legacy-platform", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "legacyplatform",
	Requires: []*analysis.Analyzer{modulejs.Analyzer, sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{legacyPlatform},
}

var legacyDetectionRegexes = []*regexp.Regexp{
	// regexp.MustCompile(`['"](app/core/.*?)|(app/plugins/.*?)['"]`),
	regexp.MustCompile(`['"](app/core/utils/promiseToDigest)|(app/plugins/.*?)|(app/core/core_module)|(app/core/core_module)|(app/core/core)['"]`),
	regexp.MustCompile(`from\s+['"]grafana\/app\/`),
	regexp.MustCompile(`System\.register\(`),
}

func run(pass *analysis.Pass) (interface{}, error) {
	moduleJsMap, ok := pass.ResultOf[modulejs.Analyzer].(*map[string][]byte)
	if !ok || moduleJsMap == nil {
		return nil, nil
	}

	hasLegacyPlatform := false

	for _, content := range *moduleJsMap {
		if hasLegacyPlatform {
			break
		}
		for _, regex := range legacyDetectionRegexes {
			if regex.Match(content) {
				pass.ReportResult(pass.AnalyzerName, legacyPlatform, "module.js: uses legacy plugin platform", "The plugin uses the legacy plugin platform (e.g. angularjs). Please migrate the plugin to use the new plugins platform.")
				hasLegacyPlatform = true
				break
			}
		}
	}

	if legacyPlatform.ReportAll && !hasLegacyPlatform {
		legacyPlatform.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, legacyPlatform, "module.js: uses current plugin platform", "")
	}

	return nil, nil
}

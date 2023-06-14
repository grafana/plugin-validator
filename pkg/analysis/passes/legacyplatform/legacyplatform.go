package legacyplatform

import (
	"bytes"
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
)

var (
	legacyPlatform = &analysis.Rule{Name: "legacy-platform", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "legacyplatform",
	Requires: []*analysis.Analyzer{modulejs.Analyzer, published.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{legacyPlatform},
}

// detector implements a check to see if a plugin uses a legacy platform (Angular).
type detector interface {
	// Detect takes the content of a module.js file and returns true if the plugin is using a legacy platform (Angular).
	Detect(moduleJs []byte) bool
}

// containsBytesDetector is a detector that returns true if the file contains the "pattern" string.
type containsBytesDetector struct {
	pattern []byte
}

// Detect returns true if moduleJs contains the byte slice d.pattern.
func (d *containsBytesDetector) Detect(moduleJs []byte) bool {
	return bytes.Contains(moduleJs, d.pattern)
}

// regexDetector is a detector that returns true if the file content matches a regular expression.
type regexDetector struct {
	regex *regexp.Regexp
}

// Detect returns true if moduleJs matches the regular expression d.regex.
func (d *regexDetector) Detect(moduleJs []byte) bool {
	return d.regex.Match(moduleJs)
}

var legacyDetectors = []detector{
	&containsBytesDetector{pattern: []byte("PanelCtrl")},
	&containsBytesDetector{pattern: []byte("QueryCtrl")},
	&containsBytesDetector{pattern: []byte("app/plugins/sdk")},
	&containsBytesDetector{pattern: []byte("angular.isNumber(")},
	&containsBytesDetector{pattern: []byte("editor.html")},
	&containsBytesDetector{pattern: []byte("ctrl.annotation")},
	&containsBytesDetector{pattern: []byte("getLegacyAngularInjector")},
	&containsBytesDetector{pattern: []byte("System.register(")},

	// &regexDetector{regex: regexp.MustCompile(`['"](app/core/.*?)|(app/plugins/.*?)['"]`)},
	&regexDetector{regex: regexp.MustCompile(`['"](app/core/utils/promiseToDigest)|(app/plugins/.*?)|(app/core/core_module)['"]`)},
	&regexDetector{regex: regexp.MustCompile(`from\s+['"]grafana\/app\/`)},
}

func run(pass *analysis.Pass) (interface{}, error) {

	_, ok := pass.ResultOf[published.Analyzer].(*published.PluginStatus)

	// we don't fail published plugins for using angular
	if ok {
		legacyPlatform.Severity = analysis.Warning
	}

	moduleJsMap, ok := pass.ResultOf[modulejs.Analyzer].(map[string][]byte)
	if !ok || len(moduleJsMap) == 0 {
		return nil, nil
	}

	hasLegacyPlatform := false

	for _, content := range moduleJsMap {
		if hasLegacyPlatform {
			break
		}
		for _, detector := range legacyDetectors {
			if detector.Detect(content) {
				pass.ReportResult(pass.AnalyzerName, legacyPlatform, "module.js: uses legacy plugin platform", "The plugin uses the legacy plugin platform (AngularJS). Please migrate the plugin to use the new plugins platform.")
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

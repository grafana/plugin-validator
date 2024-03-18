package legacyplatform

import (
	"bytes"
	"encoding/json"
	"net/http"
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

type gcomPattern struct {
	Name    string
	Type    string
	Pattern string
}

func fetchDetectors() ([]detector, error) {
	resp, err := http.Get("https://grafana.com/api/plugins/angular_patterns")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var patterns []gcomPattern
	if err := json.NewDecoder(resp.Body).Decode(&patterns); err != nil {
		return nil, err
	}

	detectors := make([]detector, len(patterns))

	for i, p := range patterns {
		if p.Type == "contains" {
			detectors[i] = &containsBytesDetector{pattern: []byte(p.Pattern)}
		}
		if p.Type == "regex" {
			detectors[i] = &regexDetector{regex: regexp.MustCompile(p.Pattern)}
		}
	}

	return detectors, nil
}

func run(pass *analysis.Pass) (interface{}, error) {

	status, ok := pass.ResultOf[published.Analyzer].(*published.PluginStatus)

	if !ok {
		return nil, nil
	}

	// we don't fail published plugins for using angular
	if status.Status != "unknown" {
		legacyPlatform.Severity = analysis.Warning
	}

	moduleJsMap, ok := pass.ResultOf[modulejs.Analyzer].(map[string][]byte)
	if !ok || len(moduleJsMap) == 0 {
		return nil, nil
	}

	hasLegacyPlatform := false

	legacyDetectors, err := fetchDetectors()
	if err != nil {
		return nil, err
	}

	for _, content := range moduleJsMap {
		if hasLegacyPlatform {
			break
		}
		for _, detector := range legacyDetectors {
			// for _, detector := range legacyDetectors {
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

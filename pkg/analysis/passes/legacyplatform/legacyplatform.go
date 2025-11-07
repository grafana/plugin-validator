package legacyplatform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"

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
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Legacy Platform",
		Description: "Detects use of Angular which is deprecated.",
	},
}

// detector implements a check to see if a plugin uses a legacy platform (Angular).
type detector interface {
	// Detect takes the content of a module.js file and returns true if the plugin is using a legacy platform (Angular).
	Detect(moduleJs []byte) bool
	Pattern() string
}

// containsBytesDetector is a detector that returns true if the file contains the "pattern" string.
type containsBytesDetector struct {
	pattern []byte
}

// Detect returns true if moduleJs contains the byte slice d.pattern.
func (d *containsBytesDetector) Detect(moduleJs []byte) bool {
	return bytes.Contains(moduleJs, d.pattern)
}

func (d *containsBytesDetector) Pattern() string {
	return string(d.pattern)
}

// regexDetector is a detector that returns true if the file content matches a regular expression.
type regexDetector struct {
	regex *regexp.Regexp
}

// Detect returns true if moduleJs matches the regular expression d.regex.
func (d *regexDetector) Detect(moduleJs []byte) bool {
	return d.regex.Match(moduleJs)
}

func (d *regexDetector) Pattern() string {
	return d.regex.String()
}

type gcomPattern struct {
	Name    string
	Type    string
	Pattern string
}

var (
	cachedGcomDetectors    []detector
	cachedGcomDetectorsMux sync.Mutex
)

func fetchDetectors() ([]detector, error) {
	// Use cache to avoid hitting rate limits in GCOM
	cachedGcomDetectorsMux.Lock()
	defer cachedGcomDetectorsMux.Unlock()
	if cachedGcomDetectors != nil {
		return cachedGcomDetectors, nil
	}

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

	// Set cache for future calls
	cachedGcomDetectors = detectors
	return detectors, nil
}

func run(pass *analysis.Pass) (interface{}, error) {

	status, ok := analysis.GetResult[*published.PluginStatus](pass, published.Analyzer)

	if !ok {
		return nil, nil
	}

	// we don't fail published plugins for using angular
	if status.Status != "unknown" {
		legacyPlatform.Severity = analysis.Warning
	}

	moduleJsMap, ok := analysis.GetResult[map[string][]byte](pass, modulejs.Analyzer)
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
			if detector.Detect(content) {
				pass.ReportResult(
					pass.AnalyzerName,
					legacyPlatform,
					"module.js: Uses the legacy AngularJS plugin platform",
					fmt.Sprintf(
						"Detected usage of '%s'. Please migrate the plugin to use the new plugins platform.",
						detector.Pattern(),
					),
				)
				hasLegacyPlatform = true
				break
			}
		}
	}

	return nil, nil
}

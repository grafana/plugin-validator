package trackingscripts

import (
	"bytes"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
)

var (
	trackingScripts = &analysis.Rule{Name: "tracking-scripts", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "trackingscripts",
	Requires: []*analysis.Analyzer{modulejs.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{trackingScripts},
}

func run(pass *analysis.Pass) (interface{}, error) {

	moduleJsMap, ok := pass.ResultOf[modulejs.Analyzer].(map[string][]byte)
	if !ok || len(moduleJsMap) == 0 {
		return nil, nil
	}

	servers := []string{
		"https://www.google-analytics.com",
		"https://api-js.mixpanel.com",
		"https://mixpanel.com",
	}

	hasTrackingScripts := false

	for _, content := range moduleJsMap {
		for _, url := range servers {
			if bytes.Contains(content, []byte(url)) {
				pass.ReportResult(pass.AnalyzerName, trackingScripts, "module.js: should not include tracking scripts", "Tracking scripts are not allowed in Grafana plugins (e.g. google analytics). Please remove any usage of tracking code.")
				hasTrackingScripts = true
				break
			}
		}
	}

	if !hasTrackingScripts {
		if trackingScripts.ReportAll {
			trackingScripts.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, trackingScripts, "module.js: no tracking scripts detected", "")
		}
	}

	return nil, nil
}

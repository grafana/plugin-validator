package trackingscripts

import (
	"bytes"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
)

var (
	trackingScripts = &analysis.Rule{Name: "tracking-scripts"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "trackingscripts",
	Requires: []*analysis.Analyzer{modulejs.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{trackingScripts},
}

func run(pass *analysis.Pass) (interface{}, error) {
	module := pass.ResultOf[modulejs.Analyzer].([]byte)

	servers := []string{
		"https://www.google-analytics.com",
		"https://api-js.mixpanel.com",
		"https://mixpanel.com",
	}

	hasTrackingScripts := false
	for _, url := range servers {
		if bytes.Contains(module, []byte(url)) {
			pass.Reportf(pass.AnalyzerName, trackingScripts, "module.js: should not include tracking scripts")
			hasTrackingScripts = true
			break
		}
	}

	if !hasTrackingScripts {
		if trackingScripts.ReportAll {
			trackingScripts.Severity = analysis.OK
			pass.Reportf(pass.AnalyzerName, trackingScripts, "module.js: no tracking scripts detected")
		}
	}

	return nil, nil
}

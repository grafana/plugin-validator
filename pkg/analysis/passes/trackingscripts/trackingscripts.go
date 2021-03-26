package trackingscripts

import (
	"bytes"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
)

var Analyzer = &analysis.Analyzer{
	Name:     "trackingscripts",
	Requires: []*analysis.Analyzer{modulejs.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	module := pass.ResultOf[modulejs.Analyzer].([]byte)

	servers := []string{
		"https://www.google-analytics.com",
		"https://api-js.mixpanel.com",
		"https://mixpanel.com",
	}

	for _, url := range servers {
		if bytes.Contains(module, []byte(url)) {
			pass.Report(analysis.Diagnostic{
				Severity: analysis.Warning,
				Message:  "should not include tracking scripts",
				Context:  "module.js",
			})
			break
		}
	}

	return nil, nil
}

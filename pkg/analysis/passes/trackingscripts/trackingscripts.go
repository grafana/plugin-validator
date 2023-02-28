package trackingscripts

import (
	"bytes"
	_ "embed"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
)

var (
	trackingScripts = &analysis.Rule{Name: "tracking-scripts", Severity: analysis.Error}
)

//go:embed list.txt
var serversList string

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

	servers := getServerList()

	hasTrackingScripts := false

	for _, content := range moduleJsMap {
		for _, url := range servers {
			if len(url) > 0 && bytes.Contains(content, []byte(url)) {
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

func getServerList() []string {
	servers := strings.Split(serversList, "\n")
	// remove empty lines and starting with # from servers
	for i := 0; i < len(servers); i++ {
		if len(servers[i]) == 0 || servers[i][0] == '#' {
			servers = append(servers[:i], servers[i+1:]...)
			i--
		}
	}
	return servers
}

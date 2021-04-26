package pluginname

import (
	"encoding/json"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	humanFriendlyName = &analysis.Rule{Name: "human-friendly-name"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "pluginname",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{humanFriendlyName},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if data.ID != "" && data.Name != "" && data.ID == data.Name {
		pass.Reportf(pass.AnalyzerName, humanFriendlyName, "plugin.json: plugin name should be human-friendly")
	}

	return nil, nil
}

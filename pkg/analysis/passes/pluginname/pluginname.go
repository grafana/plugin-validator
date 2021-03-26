package pluginname

import (
	"encoding/json"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var Analyzer = &analysis.Analyzer{
	Name:     "pluginname",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if data.ID != "" && data.Name != "" && data.ID == data.Name {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Warning,
			Message:  "plugin name should be human-friendly",
			Context:  "plugin.json",
		})
	}

	return nil, nil
}

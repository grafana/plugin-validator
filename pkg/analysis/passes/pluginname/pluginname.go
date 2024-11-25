package pluginname

import (
	"encoding/json"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
)

var (
	humanFriendlyName = &analysis.Rule{Name: "human-friendly-name", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "pluginname",
	Requires: []*analysis.Analyzer{metadata.Analyzer, published.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{humanFriendlyName},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Plugin Name formatting",
		Description: "Validates the plugin ID used conforms to our naming convention.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	publishStatus, ok := pass.ResultOf[published.Analyzer].(*published.PluginStatus)

	// we don't check published plugins for naming conventions
	if ok && publishStatus.Status != "unknown" {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if data.ID != "" && data.Name != "" && data.ID == data.Name {
		pass.ReportResult(pass.AnalyzerName, humanFriendlyName, "plugin.json: plugin name should be human-friendly", "The plugin name should be human-friendly and not the same as the plugin id. The plugin name is used in the UI and should be descriptive and easy to read.")
	} else {
		if humanFriendlyName.ReportAll {
			humanFriendlyName.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, humanFriendlyName, "plugin.json: plugin name is human-friendly", "")
		}
	}

	return nil, nil
}

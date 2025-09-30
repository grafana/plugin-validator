package pluginname

import (
	"encoding/json"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
)

var (
	humanFriendlyName = &analysis.Rule{Name: "human-friendly-name", Severity: analysis.Error}
	invalidIDFormat   = &analysis.Rule{Name: "invalid-id-format", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "pluginname",
	Requires: []*analysis.Analyzer{metadata.Analyzer, published.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{humanFriendlyName, invalidIDFormat},
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
		pass.ReportResult(
			pass.AnalyzerName,
			humanFriendlyName,
			"plugin.json: plugin name should be human-friendly",
			"The plugin name should be human-friendly and not the same as the plugin id. The plugin name is used in the UI and should be descriptive and easy to read.",
		)
	}

	idParts := strings.Split(data.ID, "-")
	if len(idParts) < 3 {
		pass.ReportResult(
			pass.AnalyzerName,
			invalidIDFormat,
			"plugin.json: plugin id should follow the format org-name-type",
			"The plugin ID should be in the format org-name-type (e.g., myorg-myplugin-panel). It must have at least 3 parts separated by hyphens.",
		)
	}

	return nil, nil
}

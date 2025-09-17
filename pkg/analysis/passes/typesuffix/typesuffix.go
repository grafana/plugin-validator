package typesuffix

import (
	"encoding/json"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	pluginTypeSuffix = &analysis.Rule{Name: "plugin-type-suffix", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "typesuffix",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{pluginTypeSuffix},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Type Suffix (panel/app/datasource)",
		Description: "Ensures the plugin has a valid type specified.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if data.Type == "" {
		return nil, nil
	}

	idParts := strings.Split(data.ID, "-")

	if idParts[len(idParts)-1] != data.Type {
		pass.ReportResult(
			pass.AnalyzerName,
			pluginTypeSuffix,
			"plugin.json: plugin id should end with plugin type",
			"E.g. \"my-plugin-name\" (a data source plugin), should be: \"my-plugin-datasource\"",
		)
	}

	return nil, nil
}

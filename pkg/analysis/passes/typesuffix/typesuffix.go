package typesuffix

import (
	"encoding/json"
	"fmt"
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
		pass.ReportResult(pass.AnalyzerName, pluginTypeSuffix, "plugin.json: plugin id should end with plugin type", "E.g. \"my-plugin-name\" (a data source plugin), should be: \"my-plugin-datasource\"")
	} else {
		if pluginTypeSuffix.ReportAll {
			pluginTypeSuffix.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, pluginTypeSuffix, fmt.Sprintf("plugin.json: plugin id ends with plugin type: %s", data.Type), "")
		}
	}

	return nil, nil
}

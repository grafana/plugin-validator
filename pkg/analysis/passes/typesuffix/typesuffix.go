package typesuffix

import (
	"encoding/json"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	pluginTypeSuffix = &analysis.Rule{Name: "plugin-type-suffix"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "typesuffix",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{pluginTypeSuffix},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if data.Type == "" {
		return nil, nil
	}

	idParts := strings.Split(data.ID, "-")

	if idParts[len(idParts)-1] != data.Type {
		pass.Reportf(pluginTypeSuffix, "plugin.json: plugin id should end with plugin type")
	}

	return nil, nil
}

package logos

import (
	"encoding/json"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	logos = &analysis.Rule{Name: "logos"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "logos",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{logos},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	reportCount := 0
	if strings.TrimSpace(data.Info.Logos.Small) == "" {
		reportCount++
		pass.Reportf(pass.AnalyzerName, logos, "plugin.json: invalid empty small logo path")
	}

	if strings.TrimSpace(data.Info.Logos.Large) == "" {
		reportCount++
		pass.Reportf(pass.AnalyzerName, logos, "plugin.json: invalid empty large logo path")
	}

	if reportCount > 0 {
		return nil, nil
	}

	return data.Info.Logos, nil
}

package discoverability

import (
	"encoding/json"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	emptyDescription = &analysis.Rule{Name: "empty-description", Severity: analysis.Warning}
	emptyKeywords    = &analysis.Rule{Name: "empty-keywords", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "discoverability",
	Requires: []*analysis.Analyzer{},
	Run:      run,
	Rules:    []*analysis.Rule{},
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

	if data.Info.Description == "" {
		pass.ReportResult(pass.AnalyzerName, emptyDescription, "plugin.json: description is empty", "Consider providing a plugin description for better discoverability.")
	}

	if len(data.Info.Keywords) == 0 {
		pass.ReportResult(pass.AnalyzerName, emptyKeywords, "plugin.json: keywords are empty", "Consider providing plugin keywords for better discoverability.")
	}

	return nil, nil
}

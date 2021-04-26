package screenshots

import (
	"encoding/json"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	screenshots = &analysis.Rule{Name: "screenshots"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "screenshots",
	Run:      checkScreenshotsExist,
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Rules:    []*analysis.Rule{screenshots},
}

func checkScreenshotsExist(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if len(data.Info.Screenshots) == 0 {
		pass.Reportf(pass.AnalyzerName, screenshots, "plugin.json: should include screenshots for marketplace")
		return nil, nil
	}

	return data.Info.Screenshots, nil
}

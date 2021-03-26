package screenshots

import (
	"encoding/json"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var Analyzer = &analysis.Analyzer{
	Name:     "screenshots",
	Run:      checkScreenshotsExist,
	Requires: []*analysis.Analyzer{metadata.Analyzer},
}

func checkScreenshotsExist(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if len(data.Info.Screenshots) == 0 {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Warning,
			Message:  "should include screenshots for marketplace",
			Context:  "plugin.json",
		})
		return nil, nil
	}

	return data.Info.Screenshots, nil
}

package screenshots

import (
	"encoding/json"
	"strings"

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
	} else {
		reportCount := 0
		for _, screenshot := range data.Info.Screenshots {
			if strings.TrimSpace(screenshot.Path) == "" {
				reportCount++
				pass.Reportf(pass.AnalyzerName, screenshots, "plugin.json: invalid empty screenshot path: %q", screenshot.Name)
			}
		}

		if reportCount > 0 {
			return nil, nil
		}

		if screenshots.ReportAll {
			screenshots.Severity = analysis.OK
			pass.Reportf(pass.AnalyzerName, screenshots, "plugin.json: includes screenshots for marketplace")
		}
	}

	return data.Info.Screenshots, nil
}

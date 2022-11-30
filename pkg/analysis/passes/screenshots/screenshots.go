package screenshots

import (
	"encoding/json"
	"fmt"
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
		explanation := "Screenshots are displayed in the plugin's catalog. Please add at least one screenshot to your plugin.json."
		pass.ReportResult(pass.AnalyzerName, screenshots, "plugin.json: should include screenshots for marketplace", explanation)
		return nil, nil
	} else {
		reportCount := 0
		for _, screenshot := range data.Info.Screenshots {
			if strings.TrimSpace(screenshot.Path) == "" {
				reportCount++
				pass.ReportResult(pass.AnalyzerName, screenshots, fmt.Sprintf("plugin.json: invalid empty screenshot path: %q", screenshot.Name), "The screenshot path must not be empty.")
			}
		}

		if reportCount > 0 {
			return nil, nil
		}

		if screenshots.ReportAll {
			screenshots.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, screenshots, "plugin.json: includes screenshots for marketplace", "")
		}
	}

	return data.Info.Screenshots, nil
}

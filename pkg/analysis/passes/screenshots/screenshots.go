package screenshots

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	screenshots = &analysis.Rule{Name: "screenshots", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "screenshots",
	Run:      checkScreenshotsExist,
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Rules:    []*analysis.Rule{screenshots},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Screenshots",
		Description: "Screenshots are specified in `plugin.json` that will be used in the Grafana plugin catalog.",
	},
}

func checkScreenshotsExist(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if len(data.Info.Screenshots) == 0 {
		explanation := "Screenshots are displayed in the Plugin catalog. Please add at least one screenshot to your plugin.json."
		pass.ReportResult(pass.AnalyzerName, screenshots, "plugin.json: should include screenshots for the Plugin catalog", explanation)
		return data.Info.Screenshots, nil
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
			pass.ReportResult(pass.AnalyzerName, screenshots, "plugin.json: includes screenshots for the Plugin catalog", "")
		}
	}

	return data.Info.Screenshots, nil
}

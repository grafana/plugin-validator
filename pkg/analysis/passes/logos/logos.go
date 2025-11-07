package logos

import (
	"fmt"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
)

var (
	logos = &analysis.Rule{Name: "logos", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "logos",
	Requires: []*analysis.Analyzer{nestedmetadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{logos},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Logos",
		Description: "Detects whether the plugin includes small and large logos to display in the plugin catalog.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {

	metadatamap, ok := analysis.GetResult[nestedmetadata.Metadatamap](pass, nestedmetadata.Analyzer)
	if !ok {
		return nil, nil
	}

	reportCount := 0
	for key, data := range metadatamap {
		if strings.TrimSpace(data.Info.Logos.Small) == "" {
			reportCount++
			pass.ReportResult(
				pass.AnalyzerName,
				logos,
				fmt.Sprintf("plugin.json: invalid empty small logo path for %s", key),
				"Logo path cannot be empty",
			)
		}

		if strings.TrimSpace(data.Info.Logos.Large) == "" {
			reportCount++
			pass.ReportResult(
				pass.AnalyzerName,
				logos,
				fmt.Sprintf("plugin.json: invalid empty large logo path for %s", key),
				"Logo path cannot be empty",
			)
		}
	}

	return nil, nil

}

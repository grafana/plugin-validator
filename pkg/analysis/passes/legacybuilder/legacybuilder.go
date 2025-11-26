package legacybuilder

import (
	"fmt"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/packagejson"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
)

var (
	legacyBuilder = &analysis.Rule{Name: "legacy-builder", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "legacybuilder",
	Requires: []*analysis.Analyzer{packagejson.Analyzer, published.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{legacyBuilder},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Legacy Grafana Toolkit usage",
		Description: "Detects the usage of the not longer supported Grafana Toolkit.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {

	publishedStatus, ok := analysis.GetResult[*published.PluginStatus](pass, published.Analyzer)

	// we don't fail published plugins for using toolkit (yet)
	if ok && publishedStatus.Status != "unknown" {
		legacyBuilder.Severity = analysis.Warning
	}

	parsedJsonContent, ok := analysis.GetResult[*packagejson.PackageJson](pass, packagejson.Analyzer)

	if !ok {
		return nil, nil
	}

	scripts := parsedJsonContent.Scripts
	if len(scripts) == 0 {
		return nil, nil
	}

	for scriptName, value := range scripts {
		// if any script contains `grafana-toolkit\s` it's a legacy builder
		if strings.Contains(value, "grafana-toolkit ") {
			// It's a legacy builder
			pass.ReportResult(
				pass.AnalyzerName,
				legacyBuilder,
				"The plugin is using a legacy builder (grafana-toolkit)",
				fmt.Sprintf("Script `%s` uses grafana-toolkit. Toolkit is deprecated and will not be updated to support new releases of Grafana. Please migrate to create-plugin https://grafana.com/developers/plugin-tools/migration-guides/migrate-from-toolkit.", scriptName),
			)
		}
	}

	return nil, nil
}

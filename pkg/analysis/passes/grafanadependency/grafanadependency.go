package grafanadependency

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/Masterminds/semver/v3"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
)

var (
	invalidGrafanaDependency    = &analysis.Rule{Name: "invalid-grafana-dependency", Severity: analysis.Error}
	validGrafanaDependency      = &analysis.Rule{Name: "valid-grafana-dependency", Severity: analysis.OK}
	mismatchedGrafanaDependency = &analysis.Rule{Name: "mismatched-grafana-dependency", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "grafanadependency",
	Requires: []*analysis.Analyzer{metadata.Analyzer, nestedmetadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{invalidGrafanaDependency, validGrafanaDependency, mismatchedGrafanaDependency},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Metadata Grafana Dependency",
		Description: "Checks that dependencies.grafanaDependency in `plugin.json` is valid.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBytes, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBytes, &data); err != nil {
		// if we fail to unmarshall it means the schema is incorrect
		// we will let the metadatavalid validator handle it
		return nil, nil
	}

	_, err := semver.NewConstraint(data.Dependencies.GrafanaDependency)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			invalidGrafanaDependency,
			fmt.Sprintf("plugin.json: dependencies.grafanaDependency field has invalid or empty version constraint: %q", data.Dependencies.GrafanaDependency),
			"The plugin.json file has an invalid or empty grafanaDependency field. Please refer to the documentation for more information. https://grafana.com/docs/grafana/latest/developers/plugins/metadata/#grafanadependency",
		)
		return nil, nil
	}

	nested, _ := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	mismatched := false
	for _, path := range slices.Sorted(maps.Keys(nested)) {
		if path == nestedmetadata.MainPluginJson {
			continue
		}

		constraint := nested[path].Dependencies.GrafanaDependency
		if constraint != data.Dependencies.GrafanaDependency {
			mismatched = true
			pass.ReportResult(
				pass.AnalyzerName,
				mismatchedGrafanaDependency,
				fmt.Sprintf("%s: dependencies.grafanaDependency %q does not match root plugin.json constraint %q", path, constraint, data.Dependencies.GrafanaDependency),
				"All nested plugins must declare the same grafanaDependency constraint as the root plugin.json.",
			)
		}
	}

	if mismatched {
		return nil, nil
	}

	if validGrafanaDependency.ReportAll {
		pass.ReportResult(pass.AnalyzerName, validGrafanaDependency, "plugin.json: dependencies.grafanaDependency field is valid", "")
	}

	return nil, nil
}

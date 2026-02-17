package grafanadependency

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	invalidGrafanaDependency = &analysis.Rule{Name: "invalid-grafana-dependency", Severity: analysis.Error}
	validGrafanaDependency   = &analysis.Rule{Name: "valid-grafana-dependency", Severity: analysis.OK}
)

var Analyzer = &analysis.Analyzer{
	Name:     "grafanadependency",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{invalidGrafanaDependency, validGrafanaDependency},
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

	if validGrafanaDependency.ReportAll {
		pass.ReportResult(pass.AnalyzerName, validGrafanaDependency, "plugin.json: dependencies.grafanaDependency field is valid", "")
	}

	return nil, nil
}

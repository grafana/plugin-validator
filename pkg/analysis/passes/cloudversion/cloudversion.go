package cloudversion

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	grafanaDependencyMissingCloudPreRelease = &analysis.Rule{
		Name:     "grafana-dependency-missing-cloud-pre-release",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "cloudversion",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{grafanaDependencyMissingCloudPreRelease},
	ReadmeInfo: analysis.ReadmeInfo{
		Name: "Cloud version",
		Description: `Ensures the Grafana version specified as Grafana dependency contains a pre-release value, ` +
			`to ensure proper support in Grafana Cloud. Runs only for Grafana Labs plugins.`,
	},
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
	// Run only for "grafana" plugins for now.
	if !strings.EqualFold(data.Info.Author.Name, "grafana labs") && !strings.EqualFold(orgFromPluginID(data.ID), "grafana") {
		return nil, nil
	}
	pre := semver.Prerelease(data.Dependencies.GrafanaDependency)
	if pre == "" {
		pass.ReportResult(
			pass.AnalyzerName,
			grafanaDependencyMissingCloudPreRelease,
			fmt.Sprintf(`Grafana dependency %q has no pre-release value`, data.Dependencies.GrafanaDependency),
			fmt.Sprintf(`The value of grafanaDependency in plugin.json (%q) is missing a pre-release value. `+
				`This may make the plugin uninstallable in Grafana Cloud. `+
				`Please add "-0" as a suffix of your grafanaDependency value ("%s-0")`,
				data.Dependencies.GrafanaDependency, data.Dependencies.GrafanaDependency,
			),
		)
	}
	return nil, nil
}

func orgFromPluginID(id string) string {
	parts := strings.SplitN(id, "-", 3)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

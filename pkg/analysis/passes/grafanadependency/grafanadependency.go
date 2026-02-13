package grafanadependency

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	missingCloudPreRelease = &analysis.Rule{
		Name:     "missing-cloud-pre-release",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "grafanadependency",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingCloudPreRelease},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Grafana Dependency",
		Description: "Ensures the Grafana dependency specified in plugin.json is valid",
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
	isGrafanaLabs := strings.EqualFold(data.Info.Author.Name, "grafana labs") || strings.EqualFold(orgFromPluginID(data.ID), "grafana")
	pre := semver.Prerelease(data.Dependencies.GrafanaDependency)
	if pre == "" && isGrafanaLabs {
		// Ensure that Grafana Labs plugin specify a pre-release (-99999999999) in Grafana Dependency.
		// If the pre-release part is missing and the grafanaDependency specifies a version that's not
		// been released yet, which is often the case for Grafana Labs plugins and not community/commercial plugins,
		// the plugin won't be loaded correctly in cloud because it doesn't satisfy the Grafana dependency.
		// Example: on a Cloud instance we have Grafana 12.4.0-99999999999. This is a PRE-RELEASE of 12.4.0.
		// If the plugin specifies 12.4.0 as grafanaDependency, it's incompatible with 12.4.0-99999999999.
		// This is because 12.4.0-x (pre-release) < 12.4.0 ("stable") => the plugin can't be installed in Cloud.
		pass.ReportResult(
			pass.AnalyzerName,
			missingCloudPreRelease,
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

// orgFromPluginID extracts and returns the organization prefix from a plugin ID by splitting on the first hyphen.
// Returns an empty string if the plugin ID is empty or invalid.
func orgFromPluginID(id string) string {
	parts := strings.SplitN(id, "-", 3)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

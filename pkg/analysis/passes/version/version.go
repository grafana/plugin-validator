package version

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-version"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
)

var (
	wrongPluginVersion = &analysis.Rule{Name: "wrong-plugin-version", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "version",
	Requires: []*analysis.Analyzer{metadata.Analyzer, published.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{wrongPluginVersion},
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

	// try to parse the submission version
	pluginSubmissionVersion := data.Info.Version
	parsedSubmissionVersion, err := version.NewVersion(pluginSubmissionVersion)
	if err != nil {
		pass.ReportResult(pass.AnalyzerName,
			wrongPluginVersion,
			fmt.Sprintf("Plugin version \"%s\" is invalid.", pluginSubmissionVersion),
			fmt.Sprintf("Could not parse plugin version \"%s\". Please use a valid semver version for your plugin. See https://semver.org/.", pluginSubmissionVersion))
		return nil, nil
	}

	pluginStatus, ok := pass.ResultOf[published.Analyzer].(*published.PluginStatus)
	if !ok {
		// in case of any error getting the online status, skip this check
		return nil, nil
	}

	// if the plugin is not published, skip this check
	if pluginStatus.Status == "unknown" {
		return nil, nil
	}

	grafanaComVersion := pluginStatus.Version
	parsedGrafanaVersion, err := version.NewVersion(grafanaComVersion)
	if err != nil {
		// in case of any error parsing the online status, skip this check
		return nil, nil
	}

	if !parsedSubmissionVersion.GreaterThan(parsedGrafanaVersion) {
		pass.ReportResult(pass.AnalyzerName,
			wrongPluginVersion,
			fmt.Sprintf("Plugin version %s is invalid.", pluginSubmissionVersion),
			fmt.Sprintf("The submitted plugin version %s is not greater than the latest published version %s on grafana.com.", pluginSubmissionVersion, grafanaComVersion),
		)
		return nil, nil
	} else if wrongPluginVersion.ReportAll {
		wrongPluginVersion.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName,
			wrongPluginVersion,
			fmt.Sprintf("Valid Plugin version %s", pluginSubmissionVersion),
			"")
	}

	return nil, nil
}

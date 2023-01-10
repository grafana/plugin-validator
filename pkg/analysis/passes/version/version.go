package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/hashicorp/go-version"
)

var (
	wrongPluginVersion = &analysis.Rule{Name: "wrong-plugin-version", Severity: analysis.Error}
)

type PluginStatus struct {
	Status  string `json:"status"`
	Slug    string `json:"slug"`
	Version string `json:"version"`
}

var Analyzer = &analysis.Analyzer{
	Name:     "version",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{wrongPluginVersion},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

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

	context, canc := context.WithTimeout(context.Background(), time.Second*15)
	defer canc()

	pluginStatus, err := getPluginDataFromGrafanaCom(context, data.ID)
	if err != nil {
		// in case of any error getting the online status, skip this check
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

func getPluginDataFromGrafanaCom(context context.Context, pluginId string) (*PluginStatus, error) {
	pluginUrl := fmt.Sprintf("https://grafana.com/api/plugins/%s?version=latest", pluginId)
	// fetch content for pluginUrl
	request, err := http.NewRequestWithContext(context, "GET", pluginUrl, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	// 404 = the plugin is not yet published
	if response.StatusCode == http.StatusNotFound {
		return nil, errors.New("plugin not found")
	}

	// != 200 = something went wrong. We can't check the plugin
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong status code, expected 200 got %d", response.StatusCode)
	}

	status := PluginStatus{}
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

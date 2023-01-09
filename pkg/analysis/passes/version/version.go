package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/prettyprint"
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

	pluginStatus, err := getPluginDataFromGrafanaCom(data.ID)
	if err != nil {
		// in case of any error getting the online status, skip this check
		// we can't fail the validator beacuse the network or API could be down and
		// other checks work offline
		return nil, nil
	}

	prettyprint.Print(pluginStatus)

	return nil, nil
}

func getPluginDataFromGrafanaCom(pluginId string) (*PluginStatus, error) {
	pluginUrl := fmt.Sprintf("https://grafana.com/api/plugins/%s?version=latest", pluginId)
	// fetch content for pluginUrl
	request, err := http.NewRequest("GET", pluginUrl, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == 404 || response.StatusCode != 200 {
		// 404 = the plugin is not yet published
		// != 200 = something went wrong. We can't check the plugin
		return nil, err
	}

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	status := &PluginStatus{}

	err = json.Unmarshal(content, status)
	if err != nil {
		return nil, err
	}
	return status, nil
}

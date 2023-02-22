package published

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var Analyzer = &analysis.Analyzer{
	Name:     "published-plugin",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{},
}

type PluginStatus struct {
	Status  string `json:"status"`
	Slug    string `json:"slug"`
	Version string `json:"version"`
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

	// 15 seconds timeout to fetch data from grafana API
	context, cancelContext := context.WithTimeout(context.Background(), time.Second*15)
	defer cancelContext()
	pluginStatus, err := getPluginDataFromGrafanaCom(context, data.ID)
	if err != nil {
		// in case of any error getting the online status, skip this check
		return nil, nil
	}

	return pluginStatus, nil
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
	defer response.Body.Close()

	// 404 = the plugin is not yet published
	if response.StatusCode == http.StatusNotFound {
		return nil, errors.New("plugin not found")
	}

	// != 200 = something went wrong. We can't check the plugin
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong status code, expected 200 got %d", response.StatusCode)
	}

	status := PluginStatus{}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

package codediff

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func mockGrafanaClockPanelVersionsAPI() {
	mockResponse := `{
					  "items": [
						{
						  "id": 5942,
						  "pluginSlug": "grafana-clock-panel",
						  "version": "2.1.7"
						},
						{
						  "id": 5942,
						  "pluginSlug": "grafana-clock-panel",
						  "version": "2.1.6"
						}
					  ],
					  "orderBy": "version",
					  "direction": "desc",
					  "pluginSlugOrId": "grafana-clock-panel",
					  "links": [
						{
						  "rel": "self",
						  "href": "/plugins/grafana-clock-panel/versions/"
						}
					  ]
					}`

	httpmock.RegisterResponder(
		"GET",
		"https://grafana.com/api/plugins/grafana-clock-panel/versions",
		httpmock.NewStringResponder(http.StatusOK, mockResponse),
	)

	// Mock GitHub API releases
	githubReleasesResponse := `[
		{
			"tag_name": "v2.1.7",
			"target_commitish": "0618b305d0c9bfe9e229ce441a90c0eec03640ba",
			"html_url": "https://github.com/grafana/clock-panel/releases/tag/v2.1.7",
			"created_at": "2022-12-01T00:00:00Z"
		},
		{
			"tag_name": "v2.1.6",
			"target_commitish": "abb44ed5bb37b9feb5e6aa64fc3b8d4bfaaf9231",
			"html_url": "https://github.com/grafana/clock-panel/releases/tag/v2.1.6",
			"created_at": "2022-11-01T00:00:00Z"
		}
	]`

	httpmock.RegisterResponder(
		"GET",
		"https://api.github.com/repos/grafana/clock-panel/releases",
		httpmock.NewStringResponder(http.StatusOK, githubReleasesResponse),
	)
}

func TestValidDiffUrlGenerated(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	pluginId := "grafana-clock-panel"
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.8"
		}
	}`)
	mockGrafanaClockPanelVersionsAPI()
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir:       "",
			SourceCodeReference: "https://github.com/grafana/clock-panel/",
		},
		ResultOf: map[*analysis.Analyzer]any{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, nil, result)
}

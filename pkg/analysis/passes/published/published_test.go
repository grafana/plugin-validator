package published

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

const testPluginId = "test-plugin-panel"

func getMockVersionResponse(id string, version string) string {
	content := fmt.Sprintf(`
	{
  	"status": "active",
  	"id": 31,
  	"typeId": 3,
  	"typeName": "Panel",
  	"typeCode": "panel",
  	"slug": "%s",
  	"name": "Clock",
  	"description": "Clock panel for grafana",
  	"version": "%s",
  	"orgName": "Grafana Labs",
  	"orgSlug": "grafana",
  	"orgUrl": "https://grafana.org",
  	"url": "https://github.com/grafana/clock-panel/",
  	"createdAt": "2016-03-31T13:09:33.000Z",
  	"updatedAt": "2023-01-04T10:24:26.000Z"
  }
	`, id, version)
	return content
}

func setupTestAnalyzer(pluginGrafanaComVersion string) (*analysis.Pass, *testpassinterceptor.TestPassInterceptor, func()) {
	var interceptor testpassinterceptor.TestPassInterceptor

	httpmock.Activate()

	responseContent := getMockVersionResponse(testPluginId, pluginGrafanaComVersion)
	responseCode := http.StatusOK

	// mock grafana.com response
	pluginUrl := fmt.Sprintf("https://grafana.com/api/plugins/%s?version=latest", testPluginId)
	httpmock.RegisterResponder("GET", pluginUrl,
		httpmock.NewStringResponder(responseCode, responseContent))

	pluginJsonContent := []byte(`{
		"id": "` + testPluginId + `",
		"type": "panel",
		"executable": "test-plugin-panel",
		"info": {
			"version": "` + pluginGrafanaComVersion + `"
		}
	}`)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}
	return pass, &interceptor, httpmock.DeactivateAndReset
}

// an unpublished plugin should simply skip this check by
// returning nil and no errors
func TestUnpublishedPlugin(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	pass, interceptor, cleanup := setupTestAnalyzer("1.0.0")
	defer cleanup()

	// mock unpublished plugin
	responseContent := `{
  	"code": "NotFound",
  	"message": "plugin not found",
  	"requestId": "mock-test-id"
	}`
	responseCode := http.StatusNotFound

	// mock grafana.com response
	pluginUrl := fmt.Sprintf("https://grafana.com/api/plugins/%s?version=latest", testPluginId)
	httpmock.RegisterResponder("GET", pluginUrl,
		httpmock.NewStringResponder(responseCode, responseContent))

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, httpmock.GetCallCountInfo(), 1)
	require.Len(t, interceptor.Diagnostics, 0)
	require.Nil(t, analyzerResult)
}

func TestPublishedPlugin(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	pass, interceptor, cleanup := setupTestAnalyzer("1.0.0")
	defer cleanup()

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	expectedStatus := &PluginStatus{
		Status:  "active",
		Slug:    "test-plugin-panel",
		Version: "1.0.0",
	}

	require.Len(t, httpmock.GetCallCountInfo(), 1)
	require.Len(t, interceptor.Diagnostics, 0)
	require.Equal(t, analyzerResult, expectedStatus)
}

package version

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

func setupTestAnalyzer(pluginSubmissionVersion string, pluginGrafanaComVersion string) (*analysis.Pass, *testpassinterceptor.TestPassInterceptor, func()) {
	var interceptor testpassinterceptor.TestPassInterceptor

	if pluginGrafanaComVersion != "" {
		httpmock.Activate()

		responseContent := getMockVersionResponse(testPluginId, pluginGrafanaComVersion)
		responseCode := http.StatusOK

		// mock grafana.com response
		pluginUrl := fmt.Sprintf("https://grafana.com/api/plugins/%s?version=latest", testPluginId)
		httpmock.RegisterResponder("GET", pluginUrl,
			httpmock.NewStringResponder(responseCode, responseContent))
	}

	pluginJsonContent := []byte(`{
		"id": "` + testPluginId + `",
		"type": "panel",
		"executable": "test-plugin-panel",
		"info": {
			"version": "` + pluginSubmissionVersion + `"
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

	pass, interceptor, cleanup := setupTestAnalyzer("1.0.0", "")
	defer cleanup()

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

func TestHigherVersion(t *testing.T) {

	pluginSubmissionVersion := "1.0.1" // version in submitted plugin.json
	pluginGrafanaComVersion := "1.0.0" // version in grafana.com
	pass, interceptor, cleanup := setupTestAnalyzer(pluginSubmissionVersion, pluginGrafanaComVersion)
	defer cleanup()

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, httpmock.GetCallCountInfo(), 1)
	require.Len(t, interceptor.Diagnostics, 0)
	require.Nil(t, analyzerResult)
}

func TestSameVersion(t *testing.T) {

	pluginSubmissionVersion := "1.0.0" // version in submitted plugin.json
	pluginGrafanaComVersion := "1.0.0" // version in grafana.com
	pass, interceptor, cleanup := setupTestAnalyzer(pluginSubmissionVersion, pluginGrafanaComVersion)
	defer cleanup()

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, httpmock.GetCallCountInfo(), 1)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Plugin version 1.0.0 is invalid.", interceptor.Diagnostics[0].Title)
	require.Equal(t, "The submitted plugin version 1.0.0 is not greater than the latest published version 1.0.0 on grafana.com.", interceptor.Diagnostics[0].Detail)
	require.Nil(t, analyzerResult)
}

func TestLowerVersion(t *testing.T) {

	pluginSubmissionVersion := "0.9.6" // version in submitted plugin.json
	pluginGrafanaComVersion := "1.0.0" // version in grafana.com
	pass, interceptor, cleanup := setupTestAnalyzer(pluginSubmissionVersion, pluginGrafanaComVersion)
	defer cleanup()

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, httpmock.GetCallCountInfo(), 1)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Plugin version 0.9.6 is invalid.", interceptor.Diagnostics[0].Title)
	require.Equal(t, "The submitted plugin version 0.9.6 is not greater than the latest published version 1.0.0 on grafana.com.", interceptor.Diagnostics[0].Detail)
	require.Nil(t, analyzerResult)
}

func TestWrongVersionFormat(t *testing.T) {

	pluginSubmissionVersion := "first-one" // version in submitted plugin.json
	pluginGrafanaComVersion := "1.0.0"     // version in grafana.com
	pass, interceptor, cleanup := setupTestAnalyzer(pluginSubmissionVersion, pluginGrafanaComVersion)
	defer cleanup()

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, httpmock.GetCallCountInfo(), 1)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Plugin version first-one is invalid.", interceptor.Diagnostics[0].Title)
	require.Equal(t, "Could not parse plugin version \"first-one\". Please use a valid semver version for your plugin. See https://semver.org/.", interceptor.Diagnostics[0].Detail)
	require.Nil(t, analyzerResult)
}

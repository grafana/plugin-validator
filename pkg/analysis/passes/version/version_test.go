package version

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

const testPluginId = "test-plugin-panel"

func setupTestAnalyzer(pluginSubmissionVersion string, pluginGrafanaComVersion string) (*analysis.Pass, *testpassinterceptor.TestPassInterceptor) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pluginJsonContent := []byte(`{
		"id": "` + testPluginId + `",
		"type": "panel",
		"executable": "test-plugin-panel",
		"info": {
			"version": "` + pluginSubmissionVersion + `"
		}
	}`)

	var pluginStatus *published.PluginStatus

	if pluginGrafanaComVersion != "" {
		pluginStatus = &published.PluginStatus{
			Status:  "active",
			Slug:    testPluginId,
			Version: pluginGrafanaComVersion,
		}
	} else {
		pluginStatus = &published.PluginStatus{
			Status: "unknown",
		}
	}

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:  pluginJsonContent,
			published.Analyzer: pluginStatus,
		},
		Report: interceptor.ReportInterceptor(),
	}
	return pass, &interceptor
}

// an unpublished plugin should simply skip this check by
// returning nil and no errors
func TestUnpublishedPlugin(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	pass, interceptor := setupTestAnalyzer("1.0.0", "")

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
	pass, interceptor := setupTestAnalyzer(pluginSubmissionVersion, pluginGrafanaComVersion)

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
	require.Nil(t, analyzerResult)
}

func TestSameVersion(t *testing.T) {

	pluginSubmissionVersion := "1.0.0" // version in submitted plugin.json
	pluginGrafanaComVersion := "1.0.0" // version in grafana.com
	pass, interceptor := setupTestAnalyzer(pluginSubmissionVersion, pluginGrafanaComVersion)

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Plugin version 1.0.0 is invalid.", interceptor.Diagnostics[0].Title)
	require.Equal(t, "The submitted plugin version 1.0.0 is not greater than the latest published version 1.0.0 on grafana.com.", interceptor.Diagnostics[0].Detail)
	require.Nil(t, analyzerResult)
}

func TestLowerVersion(t *testing.T) {

	pluginSubmissionVersion := "0.9.6" // version in submitted plugin.json
	pluginGrafanaComVersion := "1.0.0" // version in grafana.com
	pass, interceptor := setupTestAnalyzer(pluginSubmissionVersion, pluginGrafanaComVersion)

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Plugin version 0.9.6 is invalid.", interceptor.Diagnostics[0].Title)
	require.Equal(t, "The submitted plugin version 0.9.6 is not greater than the latest published version 1.0.0 on grafana.com.", interceptor.Diagnostics[0].Detail)
	require.Nil(t, analyzerResult)
}

func TestWrongVersionFormat(t *testing.T) {

	pluginSubmissionVersion := "first-one" // version in submitted plugin.json
	pluginGrafanaComVersion := "1.0.0"     // version in grafana.com
	pass, interceptor := setupTestAnalyzer(pluginSubmissionVersion, pluginGrafanaComVersion)

	analyzerResult, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Plugin version \"first-one\" is invalid.", interceptor.Diagnostics[0].Title)
	require.Equal(t, "Could not parse plugin version \"first-one\". Please use a valid semver version for your plugin. See https://semver.org/.", interceptor.Diagnostics[0].Detail)
	require.Nil(t, analyzerResult)
}

package safelinks

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

const pluginId = "test-plugin-panel"

func TestNoAPIKey(t *testing.T) {
	// Temporarily unset the API key
	originalKey := os.Getenv("WEBRISK_API_KEY")
	os.Unsetenv("WEBRISK_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("WEBRISK_API_KEY", originalKey)
		}
	}()

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(
				`{"ID": "` + pluginId + `", "info": {"links": [{"name": "Test Link", "url": "https://example.com"}]}}`,
			),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Empty(t, interceptor.Diagnostics)
}

func TestSafeLink(t *testing.T) {
	webriskApiKey := os.Getenv("WEBRISK_API_KEY")
	require.NotEmpty(t, webriskApiKey, "API key should not be empty")

	httpmock.ActivateNonDefault(httpClient)
	defer httpmock.DeactivateAndReset()

	safeURL := "https://example.com/safe-link"

	httpmock.RegisterResponder("GET", "https://webrisk.googleapis.com/v1/uris:search",
		func(req *http.Request) (*http.Response, error) {
			uri := req.URL.Query().Get("uri")
			require.Equal(t, safeURL, uri)
			return httpmock.NewStringResponder(200, `{}`)(req)
		})

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(
				`{"ID": "` + pluginId + `", "info": {"links": [{"name": "Safe Link", "url": "` + safeURL + `"}]}}`,
			),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Empty(t, interceptor.Diagnostics)
}

func TestMalwareLink(t *testing.T) {
	webriskApiKey := os.Getenv("WEBRISK_API_KEY")
	require.NotEmpty(t, webriskApiKey, "API key should not be empty")

	httpmock.ActivateNonDefault(httpClient)
	defer httpmock.DeactivateAndReset()

	malwareURL := "https://testsafebrowsing.appspot.com/s/malware_in_iframe.html"

	httpmock.RegisterResponder("GET", "https://webrisk.googleapis.com/v1/uris:search",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponder(200, `{
				"threat": {
					"threatTypes": ["MALWARE"]
				}
			}`)(req)
		})

	metadataJSON := `{"ID": "` + pluginId + `", "info": {"links": [{"name": "Malware Link", "url": "` + malwareURL + `"}]}}`

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(metadataJSON),
		},
		Report:       interceptor.ReportInterceptor(),
		AnalyzerName: "safelinks",
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Webrisk flagged link", interceptor.Diagnostics[0].Title)
	require.Contains(t, interceptor.Diagnostics[0].Detail, "Malware Link")
	require.Contains(t, interceptor.Diagnostics[0].Detail, "MALWARE")
}

func TestSocialEngineeringLink(t *testing.T) {
	webriskApiKey := os.Getenv("WEBRISK_API_KEY")
	require.NotEmpty(t, webriskApiKey, "API key should not be empty")

	httpmock.ActivateNonDefault(httpClient)
	defer httpmock.DeactivateAndReset()

	phishingURL := "https://testsafebrowsing.appspot.com/s/bad_login.html"

	httpmock.RegisterResponder("GET", "https://webrisk.googleapis.com/v1/uris:search",
		func(req *http.Request) (*http.Response, error) {
			uri := req.URL.Query().Get("uri")
			require.Equal(t, phishingURL, uri)

			return httpmock.NewStringResponder(200, `{
				"threat": {
					"threatTypes": ["SOCIAL_ENGINEERING"]
				}
			}`)(req)
		})

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(
				`{"ID": "` + pluginId + `", "info": {"links": [{"name": "Phishing Link", "url": "` + phishingURL + `"}]}}`,
			),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Webrisk flagged link", interceptor.Diagnostics[0].Title)
	require.Contains(t, interceptor.Diagnostics[0].Detail, "Phishing Link")
	require.Contains(t, interceptor.Diagnostics[0].Detail, "SOCIAL_ENGINEERING")
}

func TestMultipleThreatTypes(t *testing.T) {
	webriskApiKey := os.Getenv("WEBRISK_API_KEY")
	require.NotEmpty(t, webriskApiKey, "API key should not be empty")

	httpmock.ActivateNonDefault(httpClient)
	defer httpmock.DeactivateAndReset()

	dangerousURL := "https://example.com/very-dangerous-link"

	httpmock.RegisterResponder("GET", "https://webrisk.googleapis.com/v1/uris:search",
		func(req *http.Request) (*http.Response, error) {
			uri := req.URL.Query().Get("uri")
			require.Equal(t, dangerousURL, uri)

			return httpmock.NewStringResponder(200, `{
				"threat": {
					"threatTypes": ["MALWARE", "SOCIAL_ENGINEERING", "UNWANTED_SOFTWARE"]
				}
			}`)(req)
		})

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(
				`{"ID": "` + pluginId + `", "info": {"links": [{"name": "Dangerous Link", "url": "` + dangerousURL + `"}]}}`,
			),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Webrisk flagged link", interceptor.Diagnostics[0].Title)
	require.Contains(t, interceptor.Diagnostics[0].Detail, "Dangerous Link")
	require.Contains(t, interceptor.Diagnostics[0].Detail, "MALWARE, SOCIAL_ENGINEERING, UNWANTED_SOFTWARE")
}

func TestMultipleLinks(t *testing.T) {
	webriskApiKey := os.Getenv("WEBRISK_API_KEY")
	require.NotEmpty(t, webriskApiKey, "API key should not be empty")

	httpmock.ActivateNonDefault(httpClient)
	defer httpmock.DeactivateAndReset()

	safeURL := "https://example.com/safe"
	malwareURL := "https://example.com/malware"

	httpmock.RegisterResponder("GET", "https://webrisk.googleapis.com/v1/uris:search",
		func(req *http.Request) (*http.Response, error) {
			uri := req.URL.Query().Get("uri")

			if uri == safeURL {
				return httpmock.NewStringResponder(200, `{}`)(req)
			} else if uri == malwareURL {
				return httpmock.NewStringResponder(200, `{
					"threat": {
						"threatTypes": ["MALWARE"]
					}
				}`)(req)
			}

			return httpmock.NewStringResponder(200, `{}`)(req)
		})

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(
				`{"ID": "` + pluginId + `", "info": {"links": [
					{"name": "Safe Link", "url": "` + safeURL + `"},
					{"name": "Malware Link", "url": "` + malwareURL + `"}
				]}}`,
			),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Webrisk flagged link", interceptor.Diagnostics[0].Title)
	require.Contains(t, interceptor.Diagnostics[0].Detail, "Malware Link")
}

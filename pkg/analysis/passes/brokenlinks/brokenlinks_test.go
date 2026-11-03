package brokenlinks

import (
	"path/filepath"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

const pluginId = "test-plugin-panel"

func TestNoRelativePaths(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(`{"ID": "` + pluginId + `"}`),
			readme.Analyzer:   []byte(`# README [link](./with/relative/path)`),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Title,
		"README.md: convert relative link to absolute: ./with/relative/path",
	)
}

func TestBrokenLink(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	brokenURL := "https://example.com/broken-link"
	httpmock.RegisterResponder("GET", brokenURL,
		httpmock.NewStringResponder(404, "Not Found"))

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			metadata.Analyzer: []byte(
				`{"ID": "` + pluginId + `", "info": {"links": [{"url": "` + brokenURL + `"}]}}`,
			),
			readme.Analyzer: []byte(`# README`),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.NotEmpty(t, interceptor.Diagnostics)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"plugin.json: possible broken link: https://example.com/broken-link (404 Not Found)",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		"README.md might contain broken links. Check that all links are valid and publicly accessible.",
		interceptor.Diagnostics[0].Detail,
	)
}

func TestValidLink(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	validURL := "https://example.com/valid-link"
	httpmock.RegisterResponder("GET", validURL,
		httpmock.NewStringResponder(200, "OK"))

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			metadata.Analyzer: []byte(
				`{"ID": "` + pluginId + `", "info": {"links": [{"url": "` + validURL + `"}]}}`,
			),
			readme.Analyzer: []byte(`# README [Valid Link](` + validURL + `)`),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Empty(t, interceptor.Diagnostics)
}

func TestOwnCatalogLinkIsNotFetched(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	ownURL := "https://grafana.com/grafana/plugins/" + pluginId + "/docs/"
	httpmock.RegisterResponder("GET", ownURL,
		httpmock.NewStringResponder(404, "Not Found"))

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			metadata.Analyzer: []byte(`{"ID": "` + pluginId + `"}`),
			readme.Analyzer:   []byte(`# README [Docs](` + ownURL + `)`),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Empty(t, interceptor.Diagnostics)
	require.Zero(t, httpmock.GetTotalCallCount())
}

func TestOtherPluginCatalogLinkIsStillChecked(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	otherURL := "https://grafana.com/grafana/plugins/" + pluginId + "-other/docs/"
	httpmock.RegisterResponder("GET", otherURL,
		httpmock.NewStringResponder(404, "Not Found"))

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			metadata.Analyzer: []byte(`{"ID": "` + pluginId + `"}`),
			readme.Analyzer:   []byte(`# README [Docs](` + otherURL + `)`),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
}

func TestIsOwnCatalogURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		id   string
		want bool
	}{
		{"catalog page", "https://grafana.com/grafana/plugins/my-plugin/", "my-plugin", true},
		{"no trailing slash", "https://grafana.com/grafana/plugins/my-plugin", "my-plugin", true},
		{"docs page", "https://grafana.com/grafana/plugins/my-plugin/docs/setup/", "my-plugin", true},
		{"older form", "https://grafana.com/plugins/my-plugin", "my-plugin", true},
		{"www host", "https://www.grafana.com/grafana/plugins/my-plugin/docs/", "my-plugin", true},
		{"different case", "https://grafana.com/Grafana/Plugins/My-Plugin/", "my-plugin", true},
		{"query string", "https://grafana.com/grafana/plugins/my-plugin/?tab=docs", "my-plugin", true},
		{"id prefix of another plugin", "https://grafana.com/grafana/plugins/my-plugin-other/", "my-plugin", false},
		{"another plugin", "https://grafana.com/grafana/plugins/other/", "my-plugin", false},
		{"another host", "https://example.com/grafana/plugins/my-plugin/", "my-plugin", false},
		{"empty id", "https://grafana.com/grafana/plugins/", "", false},
		{"not a url", "::", "my-plugin", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isOwnCatalogURL(tc.url, tc.id))
		})
	}
}

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "github.com URL",
			url:      "https://github.com/grafana/grafana",
			expected: true,
		},
		{
			name:     "github.com URL with path",
			url:      "https://github.com/grafana/grafana/blob/main/README.md",
			expected: true,
		},
		{
			name:     "raw.githubusercontent.com URL",
			url:      "https://raw.githubusercontent.com/grafana/grafana/main/README.md",
			expected: false,
		},
		{
			name:     "non-GitHub URL",
			url:      "https://grafana.com",
			expected: false,
		},
		{
			name:     "another non-GitHub URL",
			url:      "https://google.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGitHubURL(tt.url)
			require.Equal(t, tt.expected, result, "Expected isGitHubURL(%s) to be %v, got %v", tt.url, tt.expected, result)
		})
	}
}

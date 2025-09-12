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

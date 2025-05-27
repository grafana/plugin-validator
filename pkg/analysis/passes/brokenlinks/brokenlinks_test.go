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

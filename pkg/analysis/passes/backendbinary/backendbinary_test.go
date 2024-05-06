package backendbinary

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestBackendFalseExecutableEmpty(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name"
  }`

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("./"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestBackendFalseExecutableWithValue(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "executable": "gpx_plugin"
  }`

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("./"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Found executable in plugin.json but backend=false",
		interceptor.Diagnostics[0].Title,
	)
}

func TestBackendTrueExecutableEmpty(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "backend": true
  }`

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("./"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Missing executable in plugin.json",
		interceptor.Diagnostics[0].Title,
	)
}

func TestAlertingTrueBackendFalse(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "alerting": true
  }`

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("./"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Found alerting in plugin.json but backend=false",
		interceptor.Diagnostics[0].Title,
	)
}

func TestBackendTrueExecutableMissing(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "backend": true,
    "executable": "gpx_plugin"
  }`

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "missing"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("testdata", "missing"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Missing backend binaries in your plugin archive",
		interceptor.Diagnostics[0].Title,
	)
}

func TestBackendTrueExecutablesFound(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "backend": true,
    "executable": "gpx_plugin"
  }`

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "missing"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("testdata", "found"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

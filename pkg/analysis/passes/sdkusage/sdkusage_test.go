package sdkusage

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestGoModNotFound(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   []byte(pluginJsonContent),
			sourcecode.Analyzer: filepath.Join("testdata", "nogomod"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"go.mod can not be found in your source code",
		interceptor.Diagnostics[0].Title,
	)
}

func TestGoModNotParseable(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   []byte(pluginJsonContent),
			sourcecode.Analyzer: filepath.Join("testdata", "gomodwrong"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"go.mod can not be parsed from your source code",
		interceptor.Diagnostics[0].Title,
	)
}

func TestValidGoMod(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   []byte(pluginJsonContent),
			sourcecode.Analyzer: filepath.Join("testdata", "validgomod"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestValidGoModWithNoGrafanaSdk(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   []byte(pluginJsonContent),
			sourcecode.Analyzer: filepath.Join("testdata", "nografanagosdk"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Your plugin uses a backend (backend=true), but the Grafana go sdk is not used",
		interceptor.Diagnostics[0].Title,
	)
}

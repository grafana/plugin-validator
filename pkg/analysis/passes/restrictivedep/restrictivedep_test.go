package restrictivedep

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestValidDependency(t *testing.T) {

	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "dependencies": {
      "grafanaDependency": ">9.0.0"
    }
  }`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestValidDependencyPatchVersion(t *testing.T) {

	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "dependencies": {
      "grafanaDependency": "9.0.x"
    }
  }`

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "plugin.json: grafanaDependency only targets patch releases of Grafana 9.0")
}

func TestValidDependencyEspecificVersion(t *testing.T) {

	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "dependencies": {
      "grafanaDependency": "9.0.1"
    }
  }`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "plugin.json: grafanaDependency only targets Grafana 9.0.1")
}

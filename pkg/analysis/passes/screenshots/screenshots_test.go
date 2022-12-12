package screenshots

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestValidScreenshots(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "info": {
      "screenshots": [
        {
          "path": "img/screenshot1.png",
          "name": "screenshot1"
        }
      ]
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

func TestNoScreenshots(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "info": {
      "screenshots": []
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
	require.Equal(t, interceptor.Diagnostics[0].Title, "plugin.json: should include screenshots for marketplace")
}

func TestInvalidScreenshotPath(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
    "name": "my plugin name",
    "info": {
      "screenshots": [{
        "path": "",
        "name": "screenshot1"
      }]
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
	require.Equal(t, interceptor.Diagnostics[0].Title, "plugin.json: invalid empty screenshot path: \"screenshot1\"")
}

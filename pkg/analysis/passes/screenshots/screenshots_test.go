package screenshots

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
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
			"path": "testdata/valid.png",
			"name": "screenshot1"
			}
		]
		}
	}`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("."),
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
			archive.Analyzer:  filepath.Join("."),
		},
		Report: interceptor.ReportInterceptor(),
	}

	res, err := Analyzer.Run(pass)
	require.Len(t, res, 0)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "plugin.json: should include screenshots for the Plugin catalog")
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
			archive.Analyzer:  filepath.Join("."),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "plugin.json: invalid empty screenshot path: \"screenshot1\"")
}

func TestInvalidScreenshotImageType(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
		"name": "my plugin name",
		"info": {
		"screenshots": [{
			"name": "enhance your existing systems",
			"path": "testdata/invalid.avif"
		}]
		}
	}`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("."),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, `invalid screenshot image type: "testdata/invalid.avif". Accepted image types: ["image/jpeg" "image/png" "image/svg+xml" "image/gif"]`, interceptor.Diagnostics[0].Title)
}

func TestTextfileScreenshotImage(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
		"name": "my plugin name",
		"info": {
		"screenshots": [
		{
			"name": "test",
			"path": "testdata/textfile.png"
		}]
		}
	}`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("."),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, `invalid screenshot image type: "testdata/textfile.png". Accepted image types: ["image/jpeg" "image/png" "image/svg+xml" "image/gif"]`, interceptor.Diagnostics[0].Title)
}

func TestJpgScreenshotImage(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
		"name": "my plugin name",
		"info": {
		"screenshots": [
		{
			"name": "test",
			"path": "testdata/valid.jpg"
		}]
		}
	}`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("."),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

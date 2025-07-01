package screenshots

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
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

func TestEmptyInvalidScreenshotPath(t *testing.T) {
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

func TestInvalidScreenshotPath(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
		"name": "my plugin name",
		"info": {
		"screenshots": [{
			"path": "testdata/helloword.png",
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
	require.Equal(t, "invalid screenshot path: \"testdata/helloword.png\"", interceptor.Diagnostics[0].Title)
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
	require.Equal(t, `invalid screenshot image: "testdata/textfile.png". Accepted image types: ["image/jpeg" "image/png" "image/gif" "image/svg+xml"]`, interceptor.Diagnostics[0].Title)
}

type tc struct {
	name              string
	pluginJsonContent string
	expectedLen       int
	expected          string
}

func TestScreenshotImageTypes(t *testing.T) {
	tcs := []tc{
		{
			name: "Valid JPG",
			pluginJsonContent: `{
				"name": "my plugin name",
				"info": {
				"screenshots": [
				{
					"name": "test",
					"path": "testdata/valid.jpg"
				}]
				}
			}`,
			expectedLen: 0,
			expected:    "",
		},
		{
			name: "Valid JPEG",
			pluginJsonContent: `{
				"name": "my plugin name",
				"info": {
				"screenshots": [
				{
					"name": "test",
					"path": "testdata/valid.jpeg"
				}]
				}
			}`,
			expectedLen: 0,
			expected:    "",
		},
		{
			name: "Valid PNG",
			pluginJsonContent: `{
				"name": "my plugin name",
				"info": {
				"screenshots": [
				{
					"name": "test",
					"path": "testdata/valid.png"
				}]
				}
			}`,
			expectedLen: 0,
			expected:    "",
		},
		{
			name: "Valid SVG text/xml",
			pluginJsonContent: `{
				"name": "my plugin name",
				"info": {
				"screenshots": [
				{
					"name": "test",
					"path": "testdata/valid.svg"
				}]
				}
			}`,
			expectedLen: 0,
			expected:    "",
		},
		{
			name: "Valid SVG text/plain",
			pluginJsonContent: `{
				"name": "my plugin name",
				"info": {
				"screenshots": [
				{
					"name": "test",
					"path": "testdata/logo.svg"
				}]
				}
			}`,
			expectedLen: 0,
			expected:    "",
		},
		{
			name: "Invalid AVIF",
			pluginJsonContent: `{
				"name": "my plugin name",
				"info": {
				"screenshots": [
				{
					"name": "test",
					"path": "testdata/invalid.avif"
				}]
				}
			}`,
			expectedLen: 1,
			expected:    `invalid screenshot image: "testdata/invalid.avif". Accepted image types: ["image/jpeg" "image/png" "image/gif" "image/svg+xml"]`,
		},
		{
			name: "Invalid WebP",
			pluginJsonContent: `{
				"name": "my plugin name",
				"info": {
				"screenshots": [
				{
					"name": "test",
					"path": "testdata/test.webp"
				}]
				}
			}`,
			expectedLen: 1,
			expected:    `invalid screenshot path: "testdata/test.webp"`,
		},
		{
			name: "Less than 512 bytes png",
			pluginJsonContent: `{
				"name": "my plugin name",
				"info": {
				"screenshots": [
				{
					"name": "test",
					"path": "testdata/small.png"
				}]
				}
			}`,
			expectedLen: 0,
			expected:    "",
		},
	}

	for _, testcase := range tcs {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			t.Logf("Running %s", testcase.name)
			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{
				RootDir: filepath.Join("./"),
				ResultOf: map[*analysis.Analyzer]interface{}{
					metadata.Analyzer: []byte(testcase.pluginJsonContent),
					archive.Analyzer:  filepath.Join("."),
				},
				Report: interceptor.ReportInterceptor(),
			}

			_, err := Analyzer.Run(pass)
			assert.NoError(t, err)
			assert.Len(t, interceptor.Diagnostics, testcase.expectedLen)
			if len(interceptor.Diagnostics) > 0 {
				assert.Equal(t, testcase.expected, interceptor.Diagnostics[0].Title)
			}
		})
	}
}

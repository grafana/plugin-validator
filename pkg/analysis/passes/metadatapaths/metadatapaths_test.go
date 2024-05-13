package metadatapaths

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/grafana/plugin-validator/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestMetadatapathsWithCorrectMetadata(t *testing.T) {
	pluginJsonContent := []byte(`{
    "id": "test-plugin-panel",
    "info": {
      "logos": {
        "small": "img/logo.svg",
        "large": "img/logo.svg"
      }
    }
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: "testdata",
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestMetadatapathsWithWrongLogoPath(t *testing.T) {
	pluginJsonContent := []byte(`{
    "id": "test-plugin-panel",
    "info": {
      "logos": {
        "small": "./img/logo.svg",
        "large": "./img/logo.svg"
      }
    }
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: "testdata",
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	require.Contains(
		t,
		interceptor.Diagnostics[0].Title,
		"plugin.json: relative small logo path should not start with",
	)
	require.Contains(
		t,
		interceptor.Diagnostics[1].Title,
		"plugin.json: relative large logo path should not start with",
	)
}

func TestMetadatapathsWithWrongScreenshotPath(t *testing.T) {
	pluginJsonContent := []byte(`{
    "id": "test-plugin-panel",
    "info": {
      "logos": {
        "small": "img/logo.svg",
        "large": "img/logo.svg"
      },
      "screenshots": [
        {
          "name": "test",
          "path": "/img/logo.png"
        },
        {
          "name": "test2",
          "path": "./img/logo.png"
        }
        ]
    }
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: "testdata",
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)

	require.Contains(
		t,
		interceptor.Diagnostics[0].Title,
		"plugin.json: relative screenshot path should not start with",
	)
	require.Contains(
		t,
		interceptor.Diagnostics[1].Title,
		"plugin.json: relative screenshot path should not start with",
	)
}

func TestMeatadapathsWithCorrectNestedLogos(t *testing.T) {
	pluginJsonContent := []byte(`{
    "id": "test-plugin-panel",
    "info": {
      "logos": {
        "small": "img/logo.svg",
        "large": "img/logo.svg"
      }
    }
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: "testdata",
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":            meta,
				"datasource/plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestMeatadapathsWithIncorrectNestedLogos(t *testing.T) {
	pluginJsonContent := []byte(`{
    "id": "test-plugin-panel",
    "info": {
      "logos": {
        "small": "img/logo.svg",
        "large": "img/logo.svg"
      }
    }
  }`)

	nestedPluginJsonContent := []byte(`{
    "id": "test-plugin-panel",
    "info": {
      "logos": {
        "small": "./img/logo.svg",
        "large": "img/logo.svg"
      }
    }
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	wrongMeta, err := utils.JSONToMetadata(nestedPluginJsonContent)
	require.NoError(t, err)

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: "testdata",
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":            meta,
				"datasource/plugin.json": wrongMeta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Contains(
		t,
		interceptor.Diagnostics[0].Title,
		"plugin.json: relative small logo path should not start with",
	)
}

package logos

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/grafana/plugin-validator/pkg/testutils"
)

const pluginId = "test-plugin-panel"

func TestValidLogos(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "id": "` + pluginId + `",
    "info": {
      "logos": {
        "small": "img/logo.svg",
        "large": "img/logo.svg"
      }
    }
  }`)

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
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

func TestEmptyLargeLogo(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "id": "` + pluginId + `",
    "info": {
      "logos": {
        "small": "img/logo.svg"
      }
    }
  }`)

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"plugin.json: invalid empty large logo path for plugin.json",
		interceptor.Diagnostics[0].Title,
	)
}

func TestEmptySmallLogo(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "id": "` + pluginId + `",
    "info": {
      "logos": {
        "large": "img/logo.svg"
      }
    }
  }`)

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"plugin.json: invalid empty small logo path for plugin.json",
		interceptor.Diagnostics[0].Title,
	)
}

func TestValidNestedLogos(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "id": "` + pluginId + `",
    "info": {
      "logos": {
        "small": "img/logo.svg",
        "large": "img/logo.svg"
      }
    }
  }`)

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{

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

func TestNestedPluginJsonMissingSmallLogo(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "id": "` + pluginId + `",
    "info": {
      "logos": {
        "small": "img/logo.svg",
        "large": "img/logo.svg"
      }
    }
  }`)

	wrongPluginJsonContent := []byte(`{
    "id": "` + pluginId + `",
    "info": {
      "logos": {
        "large": "img/logo.svg"
      }
    }
  }`)

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	wrongMeta, err := testutils.JSONToMetadata(wrongPluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{

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
	require.Equal(
		t,
		"plugin.json: invalid empty small logo path for datasource/plugin.json",
		interceptor.Diagnostics[0].Title,
	)
}

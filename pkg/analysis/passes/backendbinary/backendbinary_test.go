package backendbinary

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

func TestBackendFalseExecutableEmpty(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "test-plugin-panel",
		"type": "panel"
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("./"),
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

func TestBackendFalseExecutableWithValue(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "executable": "gpx_plugin"
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("./"),
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
		"Found executable in plugin.json but backend=false",
		interceptor.Diagnostics[0].Title,
	)
}

func TestBackendTrueExecutableEmpty(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("./"),
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
		"Missing executable in plugin.json",
		interceptor.Diagnostics[0].Title,
	)
}

func TestAlertingTrueBackendFalse(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "alerting": true
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("./"),
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
		"Found alerting in plugin.json but backend=false",
		interceptor.Diagnostics[0].Title,
	)
}

func TestBackendTrueExecutableMissing(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "missing"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "missing"),
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
		"Missing backend binaries in your plugin archive",
		interceptor.Diagnostics[0].Title,
	)
}

func TestBackendTrueExecutablesFound(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "missing"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "found"),
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

func TestBackendTrueNested(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	nestedPluginJsonContent := []byte(`{
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	nestedMeta, err := utils.JsonToMetadata(nestedPluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "missing"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "nested", "found"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":            meta,
				"datasource/plugin.json": nestedMeta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestBackendTrueOnlyNestedBinary(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name"
  }`)

	nestedPluginJsonContent := []byte(`{
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	nestedMeta, err := utils.JsonToMetadata(nestedPluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "missing"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "nested", "found"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":            meta,
				"datasource/plugin.json": nestedMeta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestBackendMissingNestedDatasource(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "executable": "gpx_plugin",
    "backend": true
  }`)

	nestedPluginJsonContent := []byte(`{
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	nestedMeta, err := utils.JsonToMetadata(nestedPluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "missing"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "nested", "missing"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":            meta,
				"datasource/plugin.json": nestedMeta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Missing backend binaries in your plugin archive",
		interceptor.Diagnostics[0].Title,
	)
}

func TestBackendFalseNested(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name"
  }`)

	nestedPluginJsonContent := []byte(`{
  }`)

	meta, err := utils.JsonToMetadata(pluginJsonContent)
	require.NoError(t, err)

	nestedMeta, err := utils.JsonToMetadata(nestedPluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "missing"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "nested", "nobinary"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":            meta,
				"datasource/plugin.json": nestedMeta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

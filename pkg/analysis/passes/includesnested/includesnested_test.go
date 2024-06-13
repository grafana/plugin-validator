package includesnested

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

func TestValidCorrectIncludesDefinition(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "test-plugin-app",
		"type": "app",
		"includes": [
      {
        "type": "datasource",
        "name": "Nested data source",
        "path": "nested-datasource/plugin.json"
      },
      {
        "type": "panel",
        "name": "nested panel",
        "path": "nested-panel/plugin.json"
      }
    ]
  }`)

	bundled1JsonContent := []byte(`{
    "id": "test-plugin-datasource",
    "type": "datasource",
    "name": "nested datasource"
  }`)

	bundled2JsonContent := []byte(`{
    "id": "test-plugin-panel",
    "type": "panel",
    "name": "nested panel"
  }`)

	mainMeta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	bundle1Meta, err := utils.JSONToMetadata(bundled1JsonContent)
	require.NoError(t, err)

	bundle2Meta, err := utils.JSONToMetadata(bundled2JsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("./"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":                   mainMeta,
				"nested-datasource/plugin.json": bundle1Meta,
				"nested-panel/plugin.json":      bundle2Meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestMissingIncludeNested(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "test-plugin-app",
		"type": "app",
		"includes": [
      {
        "type": "panel",
        "name": "nested panel",
        "path": "nested-panel/plugin.json"
      }
    ]
  }`)

	bundled1JsonContent := []byte(`{
    "id": "test-plugin-datasource",
    "type": "datasource",
    "name": "nested datasource"
  }`)

	bundled2JsonContent := []byte(`{
    "id": "test-plugin-panel",
    "type": "panel",
    "name": "nested panel"
  }`)

	mainMeta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	bundle1Meta, err := utils.JSONToMetadata(bundled1JsonContent)
	require.NoError(t, err)

	bundle2Meta, err := utils.JSONToMetadata(bundled2JsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("./"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":                   mainMeta,
				"nested-datasource/plugin.json": bundle1Meta,
				"nested-panel/plugin.json":      bundle2Meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Nested plugin nested-datasource/plugin.json is not declared in plugin main plugin.json",
		interceptor.Diagnostics[0].Title,
	)
}

func TestMissingIncludeTypeNested(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
				"id": "test-plugin-app",
		"type": "app",
		"includes": [
      {
        "type": "datasource",
        "name": "Nested data source",
        "path": "nested-datasource/plugin.json"
      },
      {
        "name": "nested panel",
        "path": "nested-panel/plugin.json"
      }
    ]
  }`)

	bundled1JsonContent := []byte(`{
    "id": "test-plugin-datasource",
    "type": "datasource",
    "name": "nested datasource"
  }`)

	//missing panel type
	bundled2JsonContent := []byte(`{
    "id": "test-plugin-panel",
    "name": "nested panel",
    "type": "panel"
  }`)

	mainMeta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	bundle1Meta, err := utils.JSONToMetadata(bundled1JsonContent)
	require.NoError(t, err)

	bundle2Meta, err := utils.JSONToMetadata(bundled2JsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("./"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":                   mainMeta,
				"nested-datasource/plugin.json": bundle1Meta,
				"nested-panel/plugin.json":      bundle2Meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Nested plugin nested-panel/plugin.json is missing type",
		interceptor.Diagnostics[0].Title,
	)
}

func TestIncludedNestedTypeMissmatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "test-plugin-app",
		"type": "app",
		"includes": [
      {
        "type": "datasource",
        "name": "Nested data source",
        "path": "nested-datasource/plugin.json"
      },
      {
        "type": "panel",
        "name": "nested panel",
        "path": "nested-panel/plugin.json"
      }
    ]
  }`)

	bundled1JsonContent := []byte(`{
    "id": "test-plugin-datasource",
    "type": "panel",
    "name": "nested datasource"
  }`)

	bundled2JsonContent := []byte(`{
    "id": "test-plugin-panel",
    "type": "panel",
    "name": "nested panel"
  }`)

	mainMeta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	bundle1Meta, err := utils.JSONToMetadata(bundled1JsonContent)
	require.NoError(t, err)

	bundle2Meta, err := utils.JSONToMetadata(bundled2JsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("./"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json":                   mainMeta,
				"nested-datasource/plugin.json": bundle1Meta,
				"nested-panel/plugin.json":      bundle2Meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Nested plugin nested-datasource/plugin.json has a type missmatch",
		interceptor.Diagnostics[0].Title,
	)

	require.Equal(
		t,
		"Plugin nested-datasource/plugin.json declared in your main plugin.json as datasource but as panel in your main plugin.json",
		interceptor.Diagnostics[0].Detail,
	)
}

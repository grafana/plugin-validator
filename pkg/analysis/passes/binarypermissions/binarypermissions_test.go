package binarypermissions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/grafana/plugin-validator/pkg/utils"
	"github.com/stretchr/testify/require"
)

const pluginId = "test-plugin-panel"

func TestBinaryFoundWithCorrectPermissions(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "test-plugin-panel"
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	// due to CI running on linux, we need to re-create the correct permissions
	testDataContainer := filepath.Join("testdata", "correct-permissions")
	require.NoError(
		t,
		os.Chmod(filepath.Join(testDataContainer, "test-plugin-panel_linux_amd64"), 0755),
	)
	require.NoError(
		t,
		os.Chmod(filepath.Join(testDataContainer, "test-plugin-panel_darwin_amd64"), 0755),
	)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: testDataContainer,
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

func TestBinaryNotFound(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "test-plugin-panel"
	}`)
	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "no-binary"),
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
		interceptor.Diagnostics[0].Title,
		"No binary found for `executable` test-plugin-panel defined in plugin.json",
	)
}

func TestBinaryFoundNested(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "binaries/test-plugin-panel"
	}`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	// due to CI running on linux, we need to re-create the correct permissions
	testDataContainer := filepath.Join("testdata", "correct-permissions-subfolder")
	require.NoError(
		t,
		os.Chmod(
			filepath.Join(testDataContainer, "binaries", "test-plugin-panel_linux_amd64"),
			0755,
		),
	)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: testDataContainer,
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

func TestBinaryIncorrectPermissions(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "test-plugin-panel"
	}`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	// due to CI running on linux, we need to re-create the correct permissions
	testDataContainer := filepath.Join("testdata", "wrong-permissions")
	require.NoError(
		t,
		os.Chmod(filepath.Join(testDataContainer, "test-plugin-panel_linux_amd64"), 0777),
	)
	require.NoError(
		t,
		os.Chmod(filepath.Join(testDataContainer, "test-plugin-panel_darwin_amd64"), 0711),
	)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "wrong-permissions"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Title,
		"Permissions for binary executable test-plugin-panel_darwin_amd64 are incorrect (0711 found).",
	)
	require.Equal(
		t,
		interceptor.Diagnostics[1].Title,
		"Permissions for binary executable test-plugin-panel_linux_amd64 are incorrect (0777 found).",
	)
}

func TestNestedBinaryFoundWithCorrectPermissions(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "test-plugin-app"
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pluginJsonNestedContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "test-plugin-datasource"
  }`)
	nestedMeta, err := utils.JSONToMetadata(pluginJsonNestedContent)
	require.NoError(t, err)

	// due to CI running on linux, we need to re-create the correct permissions
	testDataContainer := filepath.Join("testdata", "correct-permissions-nested")
	require.NoError(
		t,
		os.Chmod(filepath.Join(testDataContainer, "test-plugin-app_linux_amd64"), 0755),
	)
	require.NoError(
		t,
		os.Chmod(
			filepath.Join(testDataContainer, "datasource/test-plugin-datasource_linux_amd64"),
			0755,
		),
	)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: testDataContainer,
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

func TestNestedBinaryFoundWithIncorrectPermissions(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "test-plugin-app"
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pluginJsonNestedContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "test-plugin-datasource"
  }`)
	nestedMeta, err := utils.JSONToMetadata(pluginJsonNestedContent)
	require.NoError(t, err)

	// due to CI running on linux, we need to re-create the correct permissions
	testDataContainer := filepath.Join("testdata", "incorrect-permissions-nested")
	require.NoError(
		t,
		os.Chmod(filepath.Join(testDataContainer, "test-plugin-app_linux_amd64"), 0755),
	)
	require.NoError(
		t,
		os.Chmod(
			filepath.Join(testDataContainer, "datasource/test-plugin-datasource_linux_amd64"),
			0711,
		),
	)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: testDataContainer,
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
		"Permissions for binary executable test-plugin-datasource_linux_amd64 are incorrect (0711 found).",
		interceptor.Diagnostics[0].Title,
	)
}

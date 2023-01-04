package binarypermissions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
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

	// due to CI running on linux, we need to re-create the correct permissions
	testDataContainer := filepath.Join("testdata", "correct-permissions")
	os.Chmod(filepath.Join(testDataContainer, "test-plugin-panel_linux_amd64"), 0755)
	os.Chmod(filepath.Join(testDataContainer, "test-plugin-panel_darwin_amd64"), 0755)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
			archive.Analyzer:  testDataContainer,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
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
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
			archive.Analyzer:  filepath.Join("testdata", "no-binary"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "No binary found for `executable` test-plugin-panel defined in plugin.json")
}

func TestBinaryFoundNested(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"executable": "binaries/test-plugin-panel"
	}`)

	// due to CI running on linux, we need to re-create the correct permissions
	testDataContainer := filepath.Join("testdata", "correct-permissions-nested")
	os.Chmod(filepath.Join(testDataContainer, "binaries", "test-plugin-panel_linux_amd64"), 0755)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
			archive.Analyzer:  testDataContainer,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
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

	// due to CI running on linux, we need to re-create the correct permissions
	testDataContainer := filepath.Join("testdata", "wrong-permissions")
	os.Chmod(filepath.Join(testDataContainer, "test-plugin-panel_linux_amd64"), 0777)
	os.Chmod(filepath.Join(testDataContainer, "test-plugin-panel_darwin_amd64"), 0711)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
			archive.Analyzer:  filepath.Join("testdata", "wrong-permissions"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(t, interceptor.Diagnostics[0].Title, "Permissions for binary executable test-plugin-panel_linux_amd64 are incorrect (0777 found).")
	require.Equal(t, interceptor.Diagnostics[1].Title, "Permissions for binary executable test-plugin-panel_darwin_amd64 are incorrect (0711 found).")
}

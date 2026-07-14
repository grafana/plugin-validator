package backendbinary

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/grafana/plugin-validator/pkg/testutils"
)

func TestBackendFalseExecutableEmpty(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "test-plugin-panel",
		"type": "panel"
  }`)

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	archiveDir := t.TempDir()
	buildGoBinary(t, filepath.Join(archiveDir, "gpx_plugin_linux_amd64"), "linux", "amd64")

	pass := &analysis.Pass{
		RootDir: archiveDir,
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: archiveDir,
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	nestedMeta, err := testutils.JSONToMetadata(nestedPluginJsonContent)
	require.NoError(t, err)

	archiveDir := t.TempDir()
	buildGoBinary(t, filepath.Join(archiveDir, "gpx_plugin_linux_amd64"), "linux", "amd64")
	buildGoBinary(t, filepath.Join(archiveDir, "datasource", "gpx_plugin_linux_amd64"), "linux", "amd64")

	pass := &analysis.Pass{
		RootDir: archiveDir,
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: archiveDir,
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	nestedMeta, err := testutils.JSONToMetadata(nestedPluginJsonContent)
	require.NoError(t, err)

	archiveDir := t.TempDir()
	buildGoBinary(t, filepath.Join(archiveDir, "datasource", "gpx_plugin_linux_amd64"), "linux", "amd64")

	pass := &analysis.Pass{
		RootDir: archiveDir,
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: archiveDir,
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	nestedMeta, err := testutils.JSONToMetadata(nestedPluginJsonContent)
	require.NoError(t, err)

	archiveDir := t.TempDir()
	buildGoBinary(t, filepath.Join(archiveDir, "gpx_plugin_linux_amd64"), "linux", "amd64")
	require.NoError(t, os.MkdirAll(filepath.Join(archiveDir, "datasource"), 0o755))

	pass := &analysis.Pass{
		RootDir: archiveDir,
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: archiveDir,
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

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	nestedMeta, err := testutils.JSONToMetadata(nestedPluginJsonContent)
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

func TestBackendTrueBinaryNotLinuxAmd64(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	archiveDir := t.TempDir()
	buildGoBinary(t, filepath.Join(archiveDir, "gpx_plugin_darwin_arm64"), "darwin", "arm64")
	buildGoBinary(t, filepath.Join(archiveDir, "gpx_plugin_windows_amd64.exe"), "windows", "amd64")

	pass := &analysis.Pass{
		RootDir: archiveDir,
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: archiveDir,
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
		"Missing linux/amd64 backend binary",
		interceptor.Diagnostics[0].Title,
	)
}

func TestBackendTrueLinuxAmd64AmongMultiplePlatforms(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	meta, err := testutils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	archiveDir := t.TempDir()
	buildGoBinary(t, filepath.Join(archiveDir, "gpx_plugin_darwin_arm64"), "darwin", "arm64")
	buildGoBinary(t, filepath.Join(archiveDir, "gpx_plugin_windows_amd64.exe"), "windows", "amd64")
	buildGoBinary(t, filepath.Join(archiveDir, "gpx_plugin_linux_amd64"), "linux", "amd64")

	pass := &analysis.Pass{
		RootDir: archiveDir,
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: archiveDir,
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

func TestBackendFrontendDoesNotSkipNestedLinuxAmd64Check(t *testing.T) {
	frontendPluginJson := []byte(`{
    "name": "my plugin name"
  }`)
	nestedPluginJson := []byte(`{
    "backend": true,
    "executable": "gpx_plugin"
  }`)

	frontendMeta, err := testutils.JSONToMetadata(frontendPluginJson)
	require.NoError(t, err)
	nestedMeta, err := testutils.JSONToMetadata(nestedPluginJson)
	require.NoError(t, err)

	archiveDir := t.TempDir()
	buildGoBinary(t, filepath.Join(archiveDir, "datasource", "gpx_plugin_darwin_arm64"), "darwin", "arm64")

	// Map iteration order is unspecified, so run repeatedly: the nested backend
	// without a linux/amd64 binary must ALWAYS be reported, never skipped
	// because the frontend entry happened to be visited first.
	for i := 0; i < 50; i++ {
		var interceptor testpassinterceptor.TestPassInterceptor
		pass := &analysis.Pass{
			RootDir: archiveDir,
			ResultOf: map[*analysis.Analyzer]interface{}{
				archive.Analyzer: archiveDir,
				nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
					"plugin.json":            frontendMeta,
					"datasource/plugin.json": nestedMeta,
				},
			},
			Report: interceptor.ReportInterceptor(),
		}

		_, err := Analyzer.Run(pass)
		require.NoError(t, err)
		require.Len(t, interceptor.Diagnostics, 1, "run %d: nested backend check was skipped", i)
		require.Equal(t, "Missing linux/amd64 backend binary", interceptor.Diagnostics[0].Title)
	}
}

// buildGoBinary cross-compiles a minimal Go program for the given GOOS/GOARCH
// so tests exercise the real build-info detection instead of relying on file
// names or empty placeholder files.
func buildGoBinary(t *testing.T, destPath, goos, goarch string) {
	t.Helper()
	modDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module testbin\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0o755))
	cmd := exec.Command("go", "build", "-o", destPath, ".")
	cmd.Dir = modDir
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build test binary: %s", out)
}

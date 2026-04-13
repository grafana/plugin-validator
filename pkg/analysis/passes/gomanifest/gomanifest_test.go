package gomanifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/prettyprint"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

var (
	pluginJSONWithBackend    = []byte(`{"backend": true}`)
	pluginJSONWithoutBackend = []byte(`{}`)
)

func TestSrcWithGoFilesNoManifest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "no-manifest", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "no-manifest", "src"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Could not find or parse Go manifest file", interceptor.Diagnostics[0].Title)
}

func TestSrcWithoutGoFiles(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "no-go-files", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "no-go-files", "src"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestCorrectManifest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "correct-manifest", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "correct-manifest", "src"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestIncorrectManifest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "incorrect-manifest", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "incorrect-manifest", "src"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Invalid Go manifest file: pkg/main.go",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		"sha256sum of pkg/main.go (5cc5c557ed62f90d091328eaa28a1c57d2869d87c735985ba04a4602644409c4) does not match manifest",
		interceptor.Diagnostics[0].Detail,
	)
}

func TestMissingFileInManifest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "missing-file-in-manifest", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "missing-file-in-manifest", "src"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Invalid Go manifest file: pkg/missing.go",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		"file pkg/missing.go is in the source code but not in the manifest",
		interceptor.Diagnostics[0].Detail,
	)
}

func TestMissingFileInSourceCode(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "missing-file-in-source-code", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "missing-file-in-source-code", "src"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Invalid Go manifest file: pkg/subdir/subfile.go",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		"pkg/subdir/subfile.go is in the manifest but not in source code",
		interceptor.Diagnostics[0].Detail,
	)
}

func TestNoBackend(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "no-manifest", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "no-manifest", "src"),
			metadata.Analyzer:   pluginJSONWithoutBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestManifestWithNodeModules(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node-modules-manifest", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "node-modules-manifest", "pkg"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestWindowsManifest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "windows-manifest", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "windows-manifest", "src"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestWindowsLineEndingsManifest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "windows-line-endings", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "windows-line-endings", "src"),
			metadata.Analyzer:   pluginJSONWithBackend,
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	prettyprint.Print(interceptor.Diagnostics)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestIsNodeModulesPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "exact node_modules", path: "node_modules", expected: true},
		{name: "root node_modules child", path: "node_modules/flatted/main.go", expected: true},
		{name: "nested node_modules child", path: "src/node_modules/flatted/main.go", expected: true},
		{name: "windows separators", path: `src\\node_modules\\flatted\\main.go`, expected: true},
		{name: "similar but different directory name", path: "node_modules2/flatted/main.go", expected: false},
		{name: "regular source path", path: "pkg/main.go", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, isNodeModulesPath(tc.path))
		})
	}
}

func TestFilterPluginGoFiles(t *testing.T) {
	sourceCodeDir := filepath.Join("repo", "plugin")
	goFiles := []string{
		filepath.Join(sourceCodeDir, "pkg", "main.go"),
		filepath.Join(sourceCodeDir, "pkg", "subdir", "worker.go"),
		filepath.Join(sourceCodeDir, "node_modules", "flatted", "main.go"),
		filepath.Join(sourceCodeDir, "src", "node_modules", "dep", "dep.go"),
	}

	filtered, err := filterPluginGoFiles(sourceCodeDir, goFiles)
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(sourceCodeDir, "pkg", "main.go"),
		filepath.Join(sourceCodeDir, "pkg", "subdir", "worker.go"),
	}, filtered)
}

func TestParseManifestFile_IgnoreListIsExplicit(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "go_plugin_build_manifest")
	manifestContent := "hash-main:pkg/main.go\n" +
		"hash-ignored:node_modules/flatted/golang/pkg/flatted/flatted.go\n" +
		"hash-nested:pkg/node_modules/unsafe.go\n"

	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0o600))

	manifest, err := parseManifestFile(manifestPath)
	require.NoError(t, err)

	_, hasIgnored := manifest["node_modules/flatted/golang/pkg/flatted/flatted.go"]
	require.False(t, hasIgnored)
	require.Equal(t, "hash-main", manifest["pkg/main.go"])
	require.Equal(t, "hash-nested", manifest["pkg/node_modules/unsafe.go"])
}

func TestParseManifestFile_IgnoreListWorksWithWindowsSeparators(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "go_plugin_build_manifest")
	manifestContent := "hash-main:pkg\\main.go\n" +
		"hash-ignored:node_modules\\flatted\\golang\\pkg\\flatted\\flatted.go\n"

	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0o600))

	manifest, err := parseManifestFile(manifestPath)
	require.NoError(t, err)

	_, hasIgnored := manifest["node_modules/flatted/golang/pkg/flatted/flatted.go"]
	require.False(t, hasIgnored)
	require.Equal(t, "hash-main", manifest["pkg/main.go"])
}

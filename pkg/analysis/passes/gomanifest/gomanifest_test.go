package gomanifest

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
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
	require.Equal(t, interceptor.Diagnostics[0].Title, "Could not find or parse Go manifest file")
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
	require.Equal(t, interceptor.Diagnostics[0].Title, "The Go build manifest does not match the source code")
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
	require.Equal(t, interceptor.Diagnostics[0].Title, "The Go build manifest does not match the source code")
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
	require.Equal(t, interceptor.Diagnostics[0].Title, "The Go build manifest does not match the source code")
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

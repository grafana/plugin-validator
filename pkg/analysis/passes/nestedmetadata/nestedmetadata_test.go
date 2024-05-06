package nestedmetadata

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestNestedMetadataMissingPluginJson(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "no-plugin-json"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "missing plugin.json")
}

func TestNestedMetadataWithPluginJson(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "with-plugin-json"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestNestedMetadataWithNestedPluginJson(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "nested"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	//cast result into Metadatamap
	resultMap, ok := result.(Metadatamap)
	if !ok {
		require.Fail(t, "result is not a Metadatamap")
	}

	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
	require.Len(t, resultMap, 3)

}

func TestNestedMetadataWithNestedPluginJsonBadFormat(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "wrongnested"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)

	_, ok := result.(Metadatamap)
	// should not be able to cast it
	require.False(t, ok)

	require.Error(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Invalid plugin.json in your archive.", interceptor.Diagnostics[0].Title)
}

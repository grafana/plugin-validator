package jssourcemap

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestModuleJsDoesNotExists(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "no-source-map", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "no-source-map", "src"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "missing module.js.map in archive", interceptor.Diagnostics[0].Title)
}

func TestModuleJsDoesNotExistsNestedPlugin(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "no-source-map-nested", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "no-source-map-nested", "src"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 2)
	messages := []string{
		"missing nested/module.js.map in archive",
		"missing nested/subnested/module.js.map in archive",
	}
	require.ElementsMatch(t, messages, interceptor.GetTitles())
}

func TestSourceMapCorrect(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "source-map-correct", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "source-map-correct", "src"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestSourceMapIncorrect(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "source-map-incorrect", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "source-map-incorrect", "src"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	fmt.Println(pass.RootDir)

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "The provided javascript/typescript source code does not match your plugin archive assets.", interceptor.Diagnostics[0].Title)
}

func TestSourceMapCorrectNested(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "source-map-correct-nested", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "source-map-correct-nested", "src"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestSourceMapIncorrectNested(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "source-map-incorrect-nested", "dist"),
			sourcecode.Analyzer: filepath.Join("testdata", "source-map-incorrect-nested", "src"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "The provided javascript/typescript source code does not match your plugin archive assets.", interceptor.Diagnostics[0].Title)
}

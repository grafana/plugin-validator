package packagejson

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

const pluginId = "test-plugin-panel"

func TestVersionSourceCodeMatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.2"
		}
	}`)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir: filepath.Join("testdata", "version-match"),
		},
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   pluginJsonContent,
			sourcecode.Analyzer: "./testdata/version-match/",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestNoPackageJson(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.2"
		}
	}`)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir: filepath.Join("testdata", "no-package-json"),
		},
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   pluginJsonContent,
			sourcecode.Analyzer: "./testdata/no-package-json/",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Could not find or parse package.json from ./testdata/no-package-json/",
		interceptor.Diagnostics[0].Title,
	)
}

func TestInvalidPackageJson(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.2"
		}
	}`)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir: filepath.Join("testdata", "invalid-package-json"),
		},
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   pluginJsonContent,
			sourcecode.Analyzer: "./testdata/invalid-package-json/",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Could not find or parse package.json from ./testdata/invalid-package-json/",
		interceptor.Diagnostics[0].Title,
	)
}

// package.json in source code sometimes use a "human json" format that allows
// for comments or trailing commas. This is not valid json, but we should still be able to parse it
// because developers are used to this format.
func TestAllowHumanJson(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.5"
		}
	}`)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir: filepath.Join("testdata", "human-package-json"),
		},
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   pluginJsonContent,
			sourcecode.Analyzer: "./testdata/human-package-json/",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestVersionMisMatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	// note the test file has version 2.1.2
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.5"
		}
	}`)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir: filepath.Join("testdata", "version-mismatch"),
		},
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:   pluginJsonContent,
			sourcecode.Analyzer: "./testdata/version-mismatch/",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"The version in package.json (2.1.2) doesn't match the version in plugin.json (2.1.5)",
		interceptor.Diagnostics[0].Title,
	)
}

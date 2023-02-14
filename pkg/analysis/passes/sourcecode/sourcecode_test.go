package sourcecode

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

const pluginId = "test-plugin-panel"

func reportAll(a *analysis.Analyzer) {
	for _, r := range a.Rules {
		r.ReportAll = true
	}
}

func undoReportAll(a *analysis.Analyzer) {
	for _, r := range a.Rules {
		r.ReportAll = false
	}
}

func TestSourceCodeNotProvided(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.2"
		}
	}`)
	pass := &analysis.Pass{
		RootDir:       filepath.Join("./"),
		SourceCodeDir: "",
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	sourceCodeDir, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
	require.Equal(t, nil, sourceCodeDir)
}

func TestSourceCodeNotProvidedReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.2"
		}
	}`)
	pass := &analysis.Pass{
		RootDir:       filepath.Join("./"),
		SourceCodeDir: "",
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})

	sourceCodeDir, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Sourcecode not provided or the provided URL  does not point to a valid source code repository", interceptor.Diagnostics[0].Title)
	require.Equal(t, nil, sourceCodeDir)
}

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
		RootDir:       filepath.Join("./"),
		SourceCodeDir: filepath.Join("testdata", "version-match"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	sourceCodeDir, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
	require.Equal(t, filepath.Join("testdata", "version-match"), sourceCodeDir)
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
		RootDir:       filepath.Join("./"),
		SourceCodeDir: filepath.Join("testdata", "no-package-json"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Could not find or parse package.json from testdata/no-package-json", interceptor.Diagnostics[0].Title)
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
		RootDir:       filepath.Join("./"),
		SourceCodeDir: filepath.Join("testdata", "invalid-package-json"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Could not find or parse package.json from testdata/invalid-package-json", interceptor.Diagnostics[0].Title)
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
		RootDir:       filepath.Join("./"),
		SourceCodeDir: filepath.Join("testdata", "human-package-json"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestVersionMissMatch(t *testing.T) {
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
		RootDir:       filepath.Join("./"),
		SourceCodeDir: filepath.Join("testdata", "version-missmatch"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "The version in package.json (2.1.2) doesn't match the version in plugin.json (2.1.5)", interceptor.Diagnostics[0].Title)
}

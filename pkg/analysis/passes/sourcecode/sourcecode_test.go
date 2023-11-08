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
	require.Equal(t, "Source code not provided or the provided URL  does not point to a valid source code repository", interceptor.Diagnostics[0].Title)
	require.Equal(t, nil, sourceCodeDir)
}

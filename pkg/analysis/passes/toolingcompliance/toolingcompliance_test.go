package toolingcompliance

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestToolingComplianceValid(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "valid"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.NotNil(t, result)

	toolingCheck := result.(*ToolingCheck)
	require.True(t, toolingCheck.HasConfigDir)
	require.True(t, toolingCheck.HasGrafanaTooling)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestToolingComplianceMissingConfigDir(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "missing-config"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.NotNil(t, result)

	toolingCheck := result.(*ToolingCheck)
	require.False(t, toolingCheck.HasConfigDir)
	require.False(t, toolingCheck.HasGrafanaTooling)

	// Should have 2 diagnostics - missing config and missing tooling
	require.Len(t, interceptor.Diagnostics, 2)

	// Check that the diagnostics are errors for new plugins
	require.Equal(t, analysis.Error, interceptor.Diagnostics[0].Severity)
	require.Equal(t, analysis.Error, interceptor.Diagnostics[1].Severity)
}

func TestToolingCompliancePublishedPluginGetsWarning(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "missing-config"),
			published.Analyzer: &published.PluginStatus{
				Status: "published",
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.NotNil(t, result)

	toolingCheck := result.(*ToolingCheck)
	require.False(t, toolingCheck.HasConfigDir)

	// Should have 2 diagnostics as warnings for published plugins
	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(t, analysis.Warning, interceptor.Diagnostics[0].Severity)
	require.Equal(t, analysis.Warning, interceptor.Diagnostics[1].Severity)
}

func TestToolingComplianceNoSourceCode(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: "",
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Nil(t, result)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestToolingComplianceHasToolingButNoConfigDir(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "has-tooling-no-config"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.NotNil(t, result)

	toolingCheck := result.(*ToolingCheck)
	require.False(t, toolingCheck.HasConfigDir)
	require.True(t, toolingCheck.HasGrafanaTooling)

	// Should have 1 diagnostic for missing config dir
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "Missing .config directory", interceptor.Diagnostics[0].Title)
}

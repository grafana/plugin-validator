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

func TestToolingComplianceFullyCompliant(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "fully-compliant"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.NotNil(t, result)

	toolingCheck := result.(*ToolingCheck)
	require.True(t, toolingCheck.HasConfigDir)
	require.True(t, toolingCheck.HasGrafanaTooling)
	require.True(t, toolingCheck.HasValidWebpackConfig)
	require.True(t, toolingCheck.HasValidTsConfig)
	require.True(t, toolingCheck.HasStandardScripts)
	require.Equal(t, 0, toolingCheck.ToolingDeviationScore)
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

	// Should have at least 2 diagnostics - missing config and missing tooling
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 2)

	// Check that the diagnostics are errors for new plugins
	hasConfigError := false
	hasToolingError := false
	for _, diag := range interceptor.Diagnostics {
		if diag.Name == "missing-config-dir" {
			hasConfigError = true
			require.Equal(t, analysis.Error, diag.Severity)
		}
		if diag.Name == "missing-grafana-tooling" {
			hasToolingError = true
			require.Equal(t, analysis.Error, diag.Severity)
		}
	}
	require.True(t, hasConfigError, "Should have missing-config-dir diagnostic")
	require.True(t, hasToolingError, "Should have missing-grafana-tooling diagnostic")
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

	// Should have diagnostics as warnings for published plugins
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 2)
	for _, diag := range interceptor.Diagnostics {
		if diag.Name == "missing-config-dir" || diag.Name == "missing-grafana-tooling" {
			require.Equal(t, analysis.Warning, diag.Severity)
		}
	}
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

	// Should have at least 1 diagnostic for missing config dir
	hasConfigError := false
	for _, diag := range interceptor.Diagnostics {
		if diag.Name == "missing-config-dir" {
			hasConfigError = true
		}
	}
	require.True(t, hasConfigError, "Should have missing-config-dir diagnostic")
}

func TestToolingComplianceCustomWebpackConfig(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "custom-webpack"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.NotNil(t, result)

	toolingCheck := result.(*ToolingCheck)
	require.True(t, toolingCheck.HasConfigDir)
	require.False(t, toolingCheck.HasValidWebpackConfig)

	// Should have a diagnostic for invalid webpack config
	hasWebpackWarning := false
	for _, diag := range interceptor.Diagnostics {
		if diag.Name == "invalid-webpack-config" {
			hasWebpackWarning = true
			require.Equal(t, analysis.Warning, diag.Severity)
		}
	}
	require.True(t, hasWebpackWarning, "Should have invalid-webpack-config diagnostic")
}

func TestToolingComplianceMissingScripts(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "missing-scripts"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.NotNil(t, result)

	toolingCheck := result.(*ToolingCheck)
	require.False(t, toolingCheck.HasStandardScripts)
	require.Contains(t, toolingCheck.MissingScripts, "test")
	require.Contains(t, toolingCheck.MissingScripts, "lint")

	// Should have a diagnostic for missing scripts
	hasScriptsWarning := false
	for _, diag := range interceptor.Diagnostics {
		if diag.Name == "missing-standard-scripts" {
			hasScriptsWarning = true
			require.Equal(t, analysis.Warning, diag.Severity)
		}
	}
	require.True(t, hasScriptsWarning, "Should have missing-standard-scripts diagnostic")
}

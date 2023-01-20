package osvscanner

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestCanRunScanner(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "golang", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.NotEqual(t, "error running osv-scanner", interceptor.Diagnostics[0].Title)
}

func TestEmptyResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "golang", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "osv-scanner successfully ran", interceptor.Diagnostics[0].Title)
}

func TestNoIssueResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "node", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "osv-scanner successfully ran", interceptor.Diagnostics[0].Title)
}

// TestCriticalSeverityResults checks for a critical severity issue
func TestCriticalSeverityResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "node", "critical"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(t, "osv-scanner detected a critical severity issue", interceptor.Diagnostics[0].Title)
	require.Equal(t, "osv-scanner detected critical severity issues", interceptor.Diagnostics[1].Title)
}

// TestHighSeverityResults checks for a high severity issue
func TestHighSeverityResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "node", "high"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(t, "osv-scanner detected a high severity issue", interceptor.Diagnostics[0].Title)
	require.Equal(t, "osv-scanner detected high severity issues", interceptor.Diagnostics[1].Title)
}

// TestModerateSeverityResults checks for a moderate severity issue
func TestModerateSeverityResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "golang", "moderate"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(t, "osv-scanner detected a moderate severity issue", interceptor.Diagnostics[0].Title)
	require.Equal(t, "osv-scanner detected moderate severity issues", interceptor.Diagnostics[1].Title)
}

// TestLowSeverityResults checks for a low severity issue
func TestLowSeverityResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "node", "low"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 4)
	require.Equal(t, "osv-scanner detected a low severity issue", interceptor.Diagnostics[0].Title)
	require.Equal(t, "osv-scanner detected low severity issues", interceptor.Diagnostics[3].Title)
}

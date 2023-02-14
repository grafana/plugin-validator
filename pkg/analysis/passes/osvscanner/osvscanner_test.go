package osvscanner

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

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

// TestCanRunScanner
func TestCanRunScanner(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "golang", "none"),
			sourcecode.Analyzer: filepath.Join("testdata", "golang", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

// TestCanRunScannerReportAll
func TestCanRunScannerReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "golang", "none"),
			sourcecode.Analyzer: filepath.Join("testdata", "golang", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)
	require.Equal(t, "Binary for osv-scanner was found in PATH", interceptor.Diagnostics[0].Title)
	require.Equal(t, "osv-scanner successfully ran", interceptor.Diagnostics[1].Title)
	require.Equal(t, "osv-scanner passed", interceptor.Diagnostics[2].Title)
}

// TestEmptyResults
func TestEmptyResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "golang", "none"),
			sourcecode.Analyzer: filepath.Join("testdata", "golang", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

// TestEmptyResultsReportAll
func TestEmptyResultsReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "golang", "none"),
			sourcecode.Analyzer: filepath.Join("testdata", "golang", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)
	require.Equal(t, "osv-scanner passed", interceptor.Diagnostics[2].Title)
}

// TestNoIssueResults
func TestNoIssueResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "none"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

// TestNoIssueResultsReportAll
func TestNoIssueResultsReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "none"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "none"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 3)
	require.Equal(t, "osv-scanner passed", interceptor.Diagnostics[2].Title)
}

// TestCriticalSeverityResults
func TestCriticalSeverityResults(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "critical"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "critical"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(t, "osv-scanner detected a critical severity issue", interceptor.Diagnostics[0].Title)
	require.Equal(t, "osv-scanner detected critical severity issues", interceptor.Diagnostics[1].Title)
}

// TestCriticalSeverityResultsReportAll checks for a critical severity issue
func TestCriticalSeverityResultsReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "critical"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "critical"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 4)
	require.Equal(t, "osv-scanner detected a critical severity issue", interceptor.Diagnostics[2].Title)
	require.Equal(t, "osv-scanner detected critical severity issues", interceptor.Diagnostics[3].Title)
}

// TestHighSeverityResultsReportAll
// high severity does not report any output, unless the report all option is enabled
func TestHighSeverityResultsReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "high"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "high"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 4)
	require.Equal(t, "osv-scanner detected a high severity issue", interceptor.Diagnostics[2].Title)
	require.Equal(t, "osv-scanner detected high severity issues", interceptor.Diagnostics[3].Title)
}

// TestModerateSeverityResultsReportAll checks for a moderate severity issue
// moderate severity does not report any output, unless the report all option is enabled
func TestModerateSeverityResultsReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "golang", "moderate"),
			sourcecode.Analyzer: filepath.Join("testdata", "golang", "moderate"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 4)
	require.Equal(t, "osv-scanner detected a moderate severity issue", interceptor.Diagnostics[2].Title)
	require.Equal(t, "osv-scanner detected moderate severity issues", interceptor.Diagnostics[3].Title)
}

// TestLowSeverityResultsReportAll checks for a low severity issue
func TestLowSeverityResultsReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "low"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "low"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 6)
	require.Equal(t, "osv-scanner detected a low severity issue", interceptor.Diagnostics[2].Title)
	require.Equal(t, "osv-scanner detected low severity issues", interceptor.Diagnostics[5].Title)
}

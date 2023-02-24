package osvscanner

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func isOSVScannerInstalled(t *testing.T) bool {
	osvScannerPath, _ := exec.LookPath("osv-scanner")
	return osvScannerPath != ""
}

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
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 2)
	messages := []string{
		"Binary for osv-scanner was found in PATH",
		"osv-scanner successfully ran",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
}

// TestEmptyResults
func TestEmptyResults(t *testing.T) {
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 2)
	messages := []string{
		"Binary for osv-scanner was found in PATH",
		"osv-scanner successfully ran",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
}

// TestNoIssueResults
func TestNoIssueResults(t *testing.T) {
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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

	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 3)
	messages := []string{
		"osv-scanner passed",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
}

// TestCriticalSeverityResults
func TestCriticalSeverityResults(t *testing.T) {
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 2)
	messages := []string{
		"osv-scanner detected a critical severity issue",
		"osv-scanner detected critical severity issues",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
}

// TestCriticalSeverityResultsReportAll checks for a critical severity issue
func TestCriticalSeverityResultsReportAll(t *testing.T) {
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 4)
	messages := []string{
		"osv-scanner detected a critical severity issue",
		"osv-scanner detected critical severity issues",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
}

// TestHighSeverityResultsReportAll
// high severity does not report any output, unless the report all option is enabled
func TestHighSeverityResultsReportAll(t *testing.T) {
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 4)
	messages := []string{
		"osv-scanner detected a high severity issue",
		"osv-scanner detected high severity issues",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
}

// TestModerateSeverityResultsReportAll checks for a moderate severity issue
// moderate severity does not report any output, unless the report all option is enabled
func TestModerateSeverityResultsReportAll(t *testing.T) {
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 4)
	messages := []string{
		"osv-scanner detected a moderate severity issue",
		"osv-scanner detected moderate severity issues",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
}

// TestLowSeverityResultsReportAll checks for a low severity issue
func TestLowSeverityResultsReportAll(t *testing.T) {
	if !isOSVScannerInstalled(t) {
		t.Skip("osv-scanner not installed, skipping test")
		return
	}
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
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 6)
	messages := []string{
		"osv-scanner detected a low severity issue",
		"osv-scanner detected low severity issues",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
	details := []string{
		"SEVERITY: LOW in package debug, vulnerable to CVE-2017-16137",
	}
	require.Subset(t, interceptor.GetDetails(), details)
}

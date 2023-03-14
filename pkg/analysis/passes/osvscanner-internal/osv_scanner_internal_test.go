package OSVScannerInternal

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

// TestOSVScannerAsLibrary
func TestOSVScannerAsLibrary(t *testing.T) {
	t.Parallel()
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
	require.Len(t, interceptor.Diagnostics, 0)

	messages := []string{
		"osv-scanner detected a moderate severity issue",
		"osv-scanner detected moderate severity issues",
	}
	foo := interceptor.GetTitles()
	require.Subset(t, foo, messages)
}

func TestOSVScannerAsLibraryReportAll(t *testing.T) {
	t.Parallel()
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
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 10)

	messages := []string{
		"osv-scanner detected a high severity issue",
		"osv-scanner detected high severity issues",
		"osv-scanner detected a moderate severity issue",
		"osv-scanner detected moderate severity issues",
		"osv-scanner detected a low severity issue",
		"osv-scanner detected low severity issues",
	}
	require.Subset(t, interceptor.GetTitles(), messages)
	foo := interceptor.GetTitles()
	require.Subset(t, foo, messages)
}

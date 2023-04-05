package osvscanner

import (
	"io"
	"os"
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

	// this results in no issues since they are filtered out
	// will need to add a new lock file that is now filtered to make this a more thorough test
	/*
		messages := []string{
			"osv-scanner detected a moderate severity issue",
			"osv-scanner detected moderate severity issues",
		}
		titles := interceptor.GetTitles()
		require.Subset(t, titles, messages)
	*/
}

func TestOSVScannerAsLibraryReportAll(t *testing.T) {
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
	titles := interceptor.GetTitles()
	require.Subset(t, titles, messages)
}

func TestOSVScannerAsLibraryNoLockfile(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "doesnotexist"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "doesnotexist"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestOSVScannerAsLibraryInvalidLockfile(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "invalid"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "invalid"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	// output goes to stderr, capture it and restore
	saveStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	_, err := Analyzer.Run(pass)
	w.Close()
	got, _ := io.ReadAll(r)
	os.Stderr = saveStderr

	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
	require.Equal(t, "Failed to determine version of not a valid yarn.lock file while parsing a yarn.lock - please report this!\n", string(got))
}

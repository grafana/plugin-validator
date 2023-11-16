package osvscanner

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/osv-scanner/pkg/models"
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
			archive.Analyzer:    filepath.Join("testdata", "node", "critical-yarn"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "critical-yarn"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)

	// this results in no issues since they are filtered out
	// will need to add a new lock file that is not filtered to make this a more thorough test
	/*
		messages := []string{
			"osv-scanner detected a moderate severity issue",
			"osv-scanner detected moderate severity issues",
		}
		titles := interceptor.GetTitles()
		require.Subset(t, titles, messages)
	*/
}

// TestOSVScannerAsLibraryReportAll
// This will perform a mocked scan that return expected results of each severity type
func TestOSVScannerAsLibraryReportAll(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "critical-yarn"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "critical-yarn"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	// Turn on ReportAll for all rules, then turn it back off at the end of the test
	reportAll(Analyzer)
	t.Cleanup(func() {
		undoReportAll(Analyzer)
	})

	actualFunction := do_scan_internal

	do_scan_internal = func(lockPath string) (models.VulnerabilityResults, error) {
		group := models.GroupInfo{IDs: []string{"CVE-2021-1234"}}
		pkg := models.PackageVulns{
			Package: models.PackageInfo{Name: "fake-package"},
			Groups:  []models.GroupInfo{group},
			Vulnerabilities: []models.Vulnerability{
				{
					ID: "CVE-2020-1234",
					Severity: []models.Severity{
						{
							Type:  models.SeverityType("critical"),
							Score: "1",
						},
					},
					DatabaseSpecific: map[string]interface{}{
						"severity": SeverityCritical,
					},
				},
				{
					ID: "CVE-2021-1234",
					Severity: []models.Severity{
						{
							Type:  models.SeverityType("high"),
							Score: "1",
						},
					},
					DatabaseSpecific: map[string]interface{}{
						"severity": SeverityHigh,
					},
				},
				{
					ID: "CVE-2022-1234",
					Severity: []models.Severity{
						{
							Type:  models.SeverityType("moderate"),
							Score: "1",
						},
					},
					DatabaseSpecific: map[string]interface{}{
						"severity": SeverityModerate,
					},
				},
				{
					ID: "CVE-2023-1234",
					Severity: []models.Severity{
						{
							Type:  models.SeverityType("low"),
							Score: "1",
						},
					},
					DatabaseSpecific: map[string]interface{}{
						"severity": SeverityLow,
					},
				},
			},
		}
		source := models.PackageSource{
			Source: models.SourceInfo{
				Path: filepath.Join("testdata", "node", "critical-yarn", "yarn.lock"),
				Type: "lockfile",
			},
			Packages: []models.PackageVulns{pkg},
		}
		vulns := models.VulnerabilityResults{Results: []models.PackageSource{source}}
		//
		// restore default
		do_scan_internal = actualFunction

		return vulns, nil
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(interceptor.Diagnostics), 9)

	messages := []string{
		"osv-scanner detected a critical severity issue",
		"osv-scanner detected critical severity issues",
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

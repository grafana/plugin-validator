package osvscanner

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/osv-scanner/pkg/models"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)


var mockedDoScanInternal = func(lockPath string) (models.VulnerabilityResults, error) {
	group := models.GroupInfo{
		IDs: []string{
			"CVE-2020-1234",
			"CVE-2021-1234",
			"CVE-2022-1234",
			"CVE-2023-1234",
		},
	}
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

	return vulns, nil
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

	// expect 2 to not be filtered out of results
	actualFunction := doScanInternal
	doScanInternal = mockedDoScanInternal

	_, err := Analyzer.Run(pass)
	// restore default
	doScanInternal = actualFunction
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)

	// this results in three issues: 1 individual critical, 1 critical summary, 1 high summary
	messages := []string{
		"osv-scanner detected a critical severity issue",
		"osv-scanner detected critical severity issues",
		"osv-scanner detected high severity issues",
	}
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

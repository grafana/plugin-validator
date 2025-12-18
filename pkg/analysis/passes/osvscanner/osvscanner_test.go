package osvscanner

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/osv-scanner/v2/pkg/models"
	"github.com/ossf/osv-schema/bindings/go/osvschema"
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
		Vulnerabilities: []osvschema.Vulnerability{
			{
				ID: "CVE-2020-1234",
				Severity: []osvschema.Severity{
					{
						Type:  osvschema.SeverityType("critical"),
						Score: "1",
					},
				},
				DatabaseSpecific: map[string]interface{}{
					"severity": SeverityCritical,
				},
			},
			{
				ID: "CVE-2021-1234",
				Severity: []osvschema.Severity{
					{
						Type:  osvschema.SeverityType("high"),
						Score: "1",
					},
				},
				DatabaseSpecific: map[string]interface{}{
					"severity": SeverityHigh,
				},
			},
			{
				ID: "CVE-2022-1234",
				Severity: []osvschema.Severity{
					{
						Type:  osvschema.SeverityType("moderate"),
						Score: "1",
					},
				},
				DatabaseSpecific: map[string]interface{}{
					"severity": SeverityModerate,
				},
			},
			{
				ID: "CVE-2023-1234",
				Severity: []osvschema.Severity{
					{
						Type:  osvschema.SeverityType("low"),
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
	require.Len(t, interceptor.Diagnostics, 4)

	// this results in four issues: 1 individual critical, 1 critical summary, 1 individual high, 1 high summary
	messages := []string{
		"osv-scanner detected a critical severity issue in package fake-package",
		"osv-scanner detected critical severity issues",
		"osv-scanner detected a high severity issue in package fake-package",
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

	_, err := Analyzer.Run(pass)

	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestOSVScannerMultiVersionNPM(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "multi-version-npm"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "multi-version-npm"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	titles := interceptor.GetTitles()
	require.Contains(t, titles, "osv-scanner detected high severity issues")

	details := interceptor.GetDetails()
	hasBodyParserVulnerability := false
	for _, detail := range details {
		if strings.Contains(detail, "body-parser") && strings.Contains(detail, "SEVERITY: HIGH") {
			hasBodyParserVulnerability = true
			break
		}
	}
	require.True(
		t,
		hasBodyParserVulnerability,
		"body-parser high severity vulnerability should be reported",
	)
}

// TestOSVScannerWhitelistedPackage verifies that whitelisted packages are filtered from results
func TestOSVScannerWhitelistedPackage(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "whitelist-playwright"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "whitelist-playwright"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	actualFunction := doScanInternal
	doScanInternal = func(lockPath string) (models.VulnerabilityResults, error) {
		group := models.GroupInfo{
			IDs: []string{
				"CVE-2024-PLAYWRIGHT-001",
			},
		}
		pkg := models.PackageVulns{
			Package: models.PackageInfo{Name: "playwright", Version: "1.55.0"},
			Groups:  []models.GroupInfo{group},
			Vulnerabilities: []osvschema.Vulnerability{
				{
					ID: "CVE-2024-PLAYWRIGHT-001",
					Severity: []osvschema.Severity{
						{
							Type:  osvschema.SeverityType("high"),
							Score: "7.5",
						},
					},
					DatabaseSpecific: map[string]interface{}{
						"severity": SeverityHigh,
					},
				},
			},
		}
		source := models.PackageSource{
			Source: models.SourceInfo{
				Path: filepath.Join(
					"testdata",
					"node",
					"whitelist-playwright",
					"package-lock.json",
				),
				Type: "lockfile",
			},
			Packages: []models.PackageVulns{pkg},
		}
		vulns := models.VulnerabilityResults{Results: []models.PackageSource{source}}
		return vulns, nil
	}

	_, err := Analyzer.Run(pass)
	doScanInternal = actualFunction
	require.NoError(t, err)

	// playwright@1.55.0 is whitelisted, so no diagnostics should be reported
	require.Len(t, interceptor.Diagnostics, 0, "playwright@1.55.0 should be filtered by whitelist")
}

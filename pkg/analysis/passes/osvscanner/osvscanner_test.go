package osvscanner

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/osv-scanner/v2/pkg/models"
	"github.com/google/osv-scanner/v2/pkg/osvscanner"
	"github.com/ossf/osv-schema/bindings/go/osvschema"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

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
		Vulnerabilities: []*osvschema.Vulnerability{
			{
				Id: "CVE-2020-1234",
				Severity: []*osvschema.Severity{
					{
						Type:  osvschema.Severity_CVSS_V3,
						Score: "9.0",
					},
				},
				DatabaseSpecific: func() *structpb.Struct {
					s, _ := structpb.NewStruct(map[string]interface{}{
						"severity": SeverityCritical,
					})
					return s
				}(),
			},
			{
				Id: "CVE-2021-1234",
				Severity: []*osvschema.Severity{
					{
						Type:  osvschema.Severity_CVSS_V3,
						Score: "7.5",
					},
				},
				DatabaseSpecific: func() *structpb.Struct {
					s, _ := structpb.NewStruct(map[string]interface{}{
						"severity": SeverityHigh,
					})
					return s
				}(),
			},
			{
				Id: "CVE-2022-1234",
				Severity: []*osvschema.Severity{
					{
						Type:  osvschema.Severity_CVSS_V3,
						Score: "5.0",
					},
				},
				DatabaseSpecific: func() *structpb.Struct {
					s, _ := structpb.NewStruct(map[string]interface{}{
						"severity": SeverityModerate,
					})
					return s
				}(),
			},
			{
				Id: "CVE-2023-1234",
				Severity: []*osvschema.Severity{
					{
						Type:  osvschema.Severity_CVSS_V3,
						Score: "3.0",
					},
				},
				DatabaseSpecific: func() *structpb.Struct {
					s, _ := structpb.NewStruct(map[string]interface{}{
						"severity": SeverityLow,
					})
					return s
				}(),
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
	t.Setenv(configPathEnv, "")
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
	t.Setenv(configPathEnv, "")
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
	t.Setenv(configPathEnv, "")
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

func TestOSVScannerIgnoresSourceConfig(t *testing.T) {
	t.Setenv(configPathEnv, "")
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "source-config"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "source-config"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	titles := interceptor.GetTitles()
	require.Contains(t, titles, "osv-scanner.toml is ignored")
	require.Contains(t, titles, "osv-scanner detected a high severity issue in package body-parser")

	details := interceptor.GetDetails()
	require.True(
		t,
		slices.ContainsFunc(details, func(detail string) bool {
			return strings.Contains(detail, "CVE-2024-45590")
		}),
		"vulnerability ignored by the source config should be reported",
	)
}

func TestOSVScannerNoPackages(t *testing.T) {
	t.Setenv(configPathEnv, "")
	sourceDir := t.TempDir()
	lockPath := filepath.Join(sourceDir, "package-lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte(`{"name":"empty","version":"1.0.0","lockfileVersion":3,"packages":{}}`), 0600))

	_, err := doScanInternal(lockPath)
	require.ErrorIs(t, err, osvscanner.ErrNoPackagesFound)

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    sourceDir,
			sourcecode.Analyzer: sourceDir,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
}

func TestOSVScannerTrustedConfig(t *testing.T) {
	t.Setenv(configPathEnv, "")
	lockPath := filepath.Join("testdata", "node", "source-config", "package-lock.json")
	baseline, err := doScanInternal(lockPath)
	require.ErrorIs(t, err, osvscanner.ErrVulnerabilitiesFound)

	var baselineIDs []string
	for _, result := range baseline.Results {
		for _, pkg := range result.Packages {
			for _, vulnerability := range pkg.Vulnerabilities {
				baselineIDs = append(baselineIDs, vulnerability.Id)
			}
		}
	}
	require.Contains(t, baselineIDs, "GHSA-qwcr-r2fm-qrc7")

	configPath := filepath.Join(t.TempDir(), sourceConfigName)
	require.NoError(t, os.WriteFile(configPath, []byte(`[[IgnoredVulns]]
id = "GHSA-qwcr-r2fm-qrc7"
reason = "Approved exception"
`), 0600))
	t.Setenv(configPathEnv, configPath)

	results, err := doScanInternal(lockPath)
	if !errors.Is(err, osvscanner.ErrVulnerabilitiesFound) {
		require.NoError(t, err)
	}

	for _, result := range results.Results {
		for _, pkg := range result.Packages {
			for _, vulnerability := range pkg.Vulnerabilities {
				require.NotEqual(t, "GHSA-qwcr-r2fm-qrc7", vulnerability.Id)
				require.NotContains(t, vulnerability.Aliases, "CVE-2024-45590")
			}
		}
	}
}

func TestOSVScannerMissingConfig(t *testing.T) {
	t.Setenv(configPathEnv, filepath.Join(t.TempDir(), sourceConfigName))

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "node", "source-config"),
			sourcecode.Analyzer: filepath.Join("testdata", "node", "source-config"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.ErrorContains(t, err, "osv-scanner failed")
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
			Vulnerabilities: []*osvschema.Vulnerability{
				{
					Id: "CVE-2024-PLAYWRIGHT-001",
					Severity: []*osvschema.Severity{
						{
							Type:  osvschema.Severity_CVSS_V3,
							Score: "7.5",
						},
					},
					DatabaseSpecific: func() *structpb.Struct {
						s, _ := structpb.NewStruct(map[string]interface{}{
							"severity": SeverityHigh,
						})
						return s
					}(),
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

package osvscanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/osv-scanner/v2/pkg/models"
	"github.com/google/osv-scanner/v2/pkg/osvscanner"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	scanningFailure = &analysis.Rule{
		Name:     "osv-scanner-failed",
		Severity: analysis.Warning,
	}
	scanningParseFailure = &analysis.Rule{
		Name:     "osv-scanner-parse-failed",
		Severity: analysis.Warning,
	}
	scanningSucceeded = &analysis.Rule{
		Name:     "osv-scanner-succeeded",
		Severity: analysis.Warning,
	}
	osvScannerCriticalSeverityDetected = &analysis.Rule{
		Name:     "osv-scanner-critical-severity-vulnerabilities-detected",
		Severity: analysis.Error,
	}
	osvScannerHighSeverityDetected = &analysis.Rule{
		Name:     "osv-scanner-high-severity-vulnerabilities-detected",
		Severity: analysis.Error,
	}
	osvScannerModerateSeverityDetected = &analysis.Rule{
		Name:     "osv-scanner-moderate-severity-vulnerabilities-detected",
		Severity: analysis.Warning,
	}
	osvScannerLowSeverityDetected = &analysis.Rule{
		Name:     "osv-scanner-low-severity-vulnerabilities-detected",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "osv-scanner",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer, archive.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		osvScannerCriticalSeverityDetected,
		osvScannerHighSeverityDetected,
		osvScannerModerateSeverityDetected,
		osvScannerLowSeverityDetected,
		scanningFailure,
		scanningParseFailure,
		scanningSucceeded,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Vulnerability Scanner",
		Description:  "Detects critical vulnerabilities in Go modules and yarn lock files.",
		Dependencies: "[osv-scanner](https://github.com/google/osv-scanner), `sourceCodeUri`",
	},
}

var scannerTypes = [...]string{
	"go.mod",
	"yarn.lock",         // YARN
	"package-lock.json", // NPM
	"pnpm-lock.yaml",    // PNPM
}

func run(pass *analysis.Pass) (interface{}, error) {
	if os.Getenv("SKIP_OSV_SCANNER") != "" {
		return nil, nil
	}

	archiveFilesPath, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok || archiveFilesPath == "" {
		return nil, nil
	}
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok || sourceCodeDir == "" {
		return nil, nil
	}

	// check for the different files we check for and run scanner on each
	// if none are detected report OK
	scanningPerformed := false
	for _, scannerType := range scannerTypes {
		lockFile := filepath.Join(sourceCodeDir, scannerType)
		if _, err := os.Stat(lockFile); err == nil {
			// perform scan
			scanningPerformed = true
			data, err := doScanInternal(lockFile)
			if err != nil {
				logme.DebugFln(
					"osv-scanner returned error (vulnerabilities found): %s",
					err.Error(),
				)
			}

			filteredResults := FilterOSVResults(data, lockFile)

			// no results means no issues currently reported
			if len(filteredResults.Results) == 0 {
				scanningSucceeded.Severity = analysis.OK
				if scanningSucceeded.ReportAll {
					pass.ReportResult(
						pass.AnalyzerName,
						scanningSucceeded,
						"osv-scanner passed",
						fmt.Sprintf("No issues for %s", scannerType))
				}
			} else {
				// provide a count for each type
				criticalSeverityCount := 0
				highSeverityCount := 0

				messagesReported := map[string]bool{}
				// iterate over results
				for _, result := range filteredResults.Results {
					for _, aPackage := range result.Packages {
						//logme.DebugFln("vulnerabilities in package: %s", aPackage.Package.Name)
						for _, aVulnerability := range aPackage.Vulnerabilities {
							aliases := strings.Join(aVulnerability.Aliases, " ")
							// make sure this key exists
							severity := "n/a"
							if aVulnerability.DatabaseSpecific != nil {
								if fields := aVulnerability.DatabaseSpecific.GetFields(); fields != nil {
									if val, ok := fields["severity"]; ok {
										severity = val.GetStringValue()
									}
								}
							}
							message := fmt.Sprintf("SEVERITY: %s in package %s, vulnerable to %s", severity, aPackage.Package.Name, aliases)
							// prevent duplicate messages
							if messagesReported[message] {
								continue
							}
							// store it
							messagesReported[message] = true
							switch severity {
							case SeverityCritical:
								title := "osv-scanner detected a critical severity issue"
								if aPackage.Package.Name != "" {
									title = fmt.Sprintf("osv-scanner detected a critical severity issue in package %s", aPackage.Package.Name)
								}
								logme.DebugFln("%s: %s", title, message)
								pass.ReportResult(
									pass.AnalyzerName,
									osvScannerCriticalSeverityDetected,
									title,
									message)
								criticalSeverityCount++
							case SeverityHigh:
								title := "osv-scanner detected a high severity issue"
								if aPackage.Package.Name != "" {
									title = fmt.Sprintf("osv-scanner detected a high severity issue in package %s", aPackage.Package.Name)
								}
								logme.DebugFln("%s: %s", title, message)
								pass.ReportResult(
									pass.AnalyzerName,
									osvScannerHighSeverityDetected,
									title,
									message)
								highSeverityCount++
							}
						}
					}
				}
				if criticalSeverityCount > 0 {
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerCriticalSeverityDetected,
						"osv-scanner detected critical severity issues",
						fmt.Sprintf("osv-scanner detected %d unique critical severity issues for lockfile: %s", criticalSeverityCount, lockFile))
				}
				if highSeverityCount > 0 {
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerHighSeverityDetected,
						"osv-scanner detected high severity issues",
						fmt.Sprintf("osv-scanner detected %d unique high severity issues for lockfile: %s", highSeverityCount, lockFile))
				}
			}
		}
	}
	if !scanningPerformed {
		// nothing to do...
		scanningSucceeded.Severity = analysis.OK
		if scanningSucceeded.ReportAll {
			pass.ReportResult(
				pass.AnalyzerName,
				scanningSucceeded,
				"osv-scanner skipped",
				"Scanning skipped: No lock files detected",
			)
		}
	}

	return nil, nil
}

var doScanInternal = func(lockPath string) (models.VulnerabilityResults, error) {
	flagged := []string{
		lockPath,
	} // your real code
	vulnResult, err := osvscanner.DoScan(osvscanner.ScannerActions{
		LockfilePaths: flagged,
	})

	// logme.DebugFln("%+v", vulnResult)
	return vulnResult, err
}
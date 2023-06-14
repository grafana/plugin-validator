package osvscanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/osv-scanner/pkg/models"
	"github.com/google/osv-scanner/pkg/osvscanner"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
)

var (
	scanningFailure                    = &analysis.Rule{Name: "osv-scanner-failed", Severity: analysis.Warning}
	scanningParseFailure               = &analysis.Rule{Name: "osv-scanner-parse-failed", Severity: analysis.Warning}
	scanningSucceeded                  = &analysis.Rule{Name: "osv-scanner-succeeded", Severity: analysis.Warning}
	osvScannerCriticalSeverityDetected = &analysis.Rule{Name: "osv-scanner-critical-severity-vulnerabilities-detected", Severity: analysis.Warning} // This will be set to Error once stable
	osvScannerHighSeverityDetected     = &analysis.Rule{Name: "osv-scanner-high-severity-vulnerabilities-detected", Severity: analysis.Warning}
	osvScannerModerateSeverityDetected = &analysis.Rule{Name: "osv-scanner-moderate-severity-vulnerabilities-detected", Severity: analysis.Warning}
	osvScannerLowSeverityDetected      = &analysis.Rule{Name: "osv-scanner-low-severity-vulnerabilities-detected", Severity: analysis.Warning}
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
		scanningSucceeded},
}

var scannerTypes = [...]string{
	"go.mod",
	"yarn.lock",         // YARN
	"package-lock.json", // NPM
	"pnpm-lock.yaml",    // PNPM
}

func run(pass *analysis.Pass) (interface{}, error) {
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
			data, err := scanInternal(lockFile)
			if err != nil {
				if scanningFailure.ReportAll {
					pass.ReportResult(
						pass.AnalyzerName,
						scanningFailure,
						"osv-scanner failed to run",
						fmt.Sprintf("osv-scanner failed to run: %s", err.Error()))
				}
			} else {
				scanningFailure.Severity = analysis.OK
				if scanningFailure.ReportAll {
					pass.ReportResult(
						pass.AnalyzerName,
						scanningFailure,
						"osv-scanner successfully ran",
						"osv-scanner successfully ran")
				}
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
				moderateSeverityCount := 0
				lowSeverityCount := 0

				messagesReported := map[string]bool{}
				// iterate over results
				for _, result := range filteredResults.Results {
					for _, aPackage := range result.Packages {
						//logme.DebugFln("vulnerabilities in package: %s", aPackage.Package.Name)
						for _, aVulnerability := range aPackage.Vulnerabilities {
							aliases := strings.Join(aVulnerability.Aliases, " ")
							// make sure this key exists
							severity := "n/a"
							if val, ok := aVulnerability.DatabaseSpecific["severity"]; ok {
								severity = val.(string)
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
								pass.ReportResult(
									pass.AnalyzerName,
									osvScannerCriticalSeverityDetected,
									"osv-scanner detected a critical severity issue",
									message)
								criticalSeverityCount++
							case SeverityHigh:
								if osvScannerHighSeverityDetected.ReportAll {
									pass.ReportResult(
										pass.AnalyzerName,
										osvScannerHighSeverityDetected,
										"osv-scanner detected a high severity issue",
										message)
								}
								highSeverityCount++
							case SeverityModerate:
								if osvScannerModerateSeverityDetected.ReportAll {
									pass.ReportResult(
										pass.AnalyzerName,
										osvScannerModerateSeverityDetected,
										"osv-scanner detected a moderate severity issue",
										message)
								}
								moderateSeverityCount++
							case SeverityLow:
								if osvScannerLowSeverityDetected.ReportAll {
									pass.ReportResult(
										pass.AnalyzerName,
										osvScannerLowSeverityDetected,
										"osv-scanner detected a low severity issue",
										message)
								}
								lowSeverityCount++
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
				if highSeverityCount > 0 && osvScannerHighSeverityDetected.ReportAll {
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerHighSeverityDetected,
						"osv-scanner detected high severity issues",
						fmt.Sprintf("osv-scanner detected %d unique high severity issues for lockfile: %s", highSeverityCount, lockFile))
				}
				if moderateSeverityCount > 0 && osvScannerModerateSeverityDetected.ReportAll {
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerModerateSeverityDetected,
						"osv-scanner detected moderate severity issues",
						fmt.Sprintf("osv-scanner detected %d unique moderate severity issues for lockfile: %s", moderateSeverityCount, lockFile))
				}
				if lowSeverityCount > 0 && osvScannerLowSeverityDetected.ReportAll {
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerLowSeverityDetected,
						"osv-scanner detected low severity issues",
						fmt.Sprintf("osv-scanner detected %d unique low severity issues for lockfile: %s", lowSeverityCount, lockFile))
				}
			}
		}
	}
	if !scanningPerformed {
		// nothing to do...
		scanningSucceeded.Severity = analysis.OK
		if scanningSucceeded.ReportAll {
			pass.ReportResult(pass.AnalyzerName, scanningSucceeded, "osv-scanner skipped", "Scanning skipped: No lock files detected")
		}
	}

	return nil, nil
}

func scanInternal(lockPath string) (models.VulnerabilityResults, error) {
	flagged := []string{
		lockPath,
	}

	vulnResult, err := osvscanner.DoScan(osvscanner.ScannerActions{
		LockfilePaths: flagged,
	}, nil)

	// logme.DebugFln("%+v", vulnResult)
	return vulnResult, err
}

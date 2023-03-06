package osvscanner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	missingOSVScanner                  = &analysis.Rule{Name: "osvscanner-missing-binary", Severity: analysis.Warning}
	scanningFailure                    = &analysis.Rule{Name: "osvscanner-exec-failed", Severity: analysis.Warning}
	scanningParseFailure               = &analysis.Rule{Name: "osvscanner-parse-failed", Severity: analysis.Warning}
	scanningSucceeded                  = &analysis.Rule{Name: "osvscanner-succeeded", Severity: analysis.Warning}
	osvScannerCriticalSeverityDetected = &analysis.Rule{Name: "osvscanner-critical-severity-vulnerabilities-detected", Severity: analysis.Warning} // This will be set to Error once stable
	osvScannerHighSeverityDetected     = &analysis.Rule{Name: "osvscanner-high-severity-vulnerabilities-detected", Severity: analysis.Warning}
	osvScannerModerateSeverityDetected = &analysis.Rule{Name: "osvscanner-moderate-severity-vulnerabilities-detected", Severity: analysis.Warning}
	osvScannerLowSeverityDetected      = &analysis.Rule{Name: "osvscanner-low-severity-vulnerabilities-detected", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "osv-scanner",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer, archive.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		missingOSVScanner,
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
	"yarn.lock",
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

	// make sure osv-scanner is in PATH
	ovsBinaryPath, err := exec.LookPath("osv-scanner")
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			missingOSVScanner,
			"Binary for osv-scanner not found in PATH", "osv-scanner executable must be in your shell PATH.")
		return nil, nil
	} else {
		missingOSVScanner.Severity = analysis.OK
		if missingOSVScanner.ReportAll {
			pass.ReportResult(
				pass.AnalyzerName,
				missingOSVScanner,
				"Binary for osv-scanner was found in PATH", "osv-scanner executable exists in your shell PATH.")
		}
	}

	// check for the different files we check for and run scanner on each
	// if none are detected report OK
	scanningPerformed := false
	for _, scannerType := range scannerTypes {
		lockFile := filepath.Join(sourceCodeDir, scannerType)
		if _, err := os.Stat(lockFile); err == nil {
			// perform scan
			scanningPerformed = true
			cmdArgs := []string{"--json", "--lockfile", lockFile}
			data, err := exec.Command(ovsBinaryPath, cmdArgs...).Output()
			// error output is expected from osv-scanner, but if the length is zero there was a problem
			// running the command
			if err != nil && len(string(err.Error())) == 0 {
				// no output to stderr is an error
				return nil, err
			}
			scanningFailure.Severity = analysis.OK
			if scanningFailure.ReportAll {
				pass.ReportResult(
					pass.AnalyzerName,
					scanningFailure,
					"osv-scanner successfully ran",
					"osv-scanner successfully ran and has output")
			}
			// deserialize json output, detect CRITICAL severity
			var objmap OSVJsonOutput
			if err := json.Unmarshal(data, &objmap); err != nil {
				pass.ReportResult(
					pass.AnalyzerName,
					scanningFailure,
					"osv-scanner output not recognized",
					fmt.Sprintf("osv-scanner output for file %s could not be parsed: %s", scannerType, err))
			}

			filteredResults := FilterOSVResults(objmap)

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
						logme.DebugFln("vulnerabilities in package: %s", aPackage.Package.Name)
						for _, aVulnerability := range aPackage.Vulnerabilities {
							aliases := strings.Join(aVulnerability.Aliases, " ")
							message := fmt.Sprintf("SEVERITY: %s in package %s, vulnerable to %s", aVulnerability.DatabaseSpecific.Severity, aPackage.Package.Name, aliases)
							// prevent duplicate messages
							if messagesReported[message] {
								continue
							}
							// store it
							messagesReported[message] = true
							switch aVulnerability.DatabaseSpecific.Severity {
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

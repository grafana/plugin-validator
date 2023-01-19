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
)

var (
	missingOSVScanner                  = &analysis.Rule{Name: "missing-osvscanner-binary", Severity: analysis.Warning}
	scanningFailure                    = &analysis.Rule{Name: "osvscanner-failed", Severity: analysis.Warning}
	scanningParseFailure               = &analysis.Rule{Name: "osvscanner-parse-failed", Severity: analysis.Warning}
	noIssues                           = &analysis.Rule{Name: "osvscanner-no-issues", Severity: analysis.OK}
	osvScannerCriticalSeverityDetected = &analysis.Rule{Name: "critical-severity-vulnerabilities-detected", Severity: analysis.Error}
	osvScannerHighSeverityDetected     = &analysis.Rule{Name: "high-severity-vulnerabilities-detected-golang", Severity: analysis.Warning}
	osvScannerModerateSeverityDetected = &analysis.Rule{Name: "moderate-severity-vulnerabilities-detected-golang", Severity: analysis.Warning}
	osvScannerLowSeverityDetected      = &analysis.Rule{Name: "low-severity-vulnerabilities-detected-golang", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "osv-scanner",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		missingOSVScanner,
		osvScannerCriticalSeverityDetected,
		osvScannerHighSeverityDetected,
		osvScannerModerateSeverityDetected,
		osvScannerLowSeverityDetected,
		scanningFailure,
		noIssues,
		scanningParseFailure},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)
	// we're detecting only go.mod and package.lock (this can be changed to use defaults)
	lockFile := filepath.Join(archiveDir, "go.mod")
	if _, err := os.Stat(lockFile); err != nil {
		// check for yarn.lock
		lockFile = filepath.Join(archiveDir, "yarn.lock")
		if _, err := os.Stat(lockFile); err != nil {
			// nothing to do... skip
			return nil, nil
		}
	}
	path, err := exec.LookPath("osv-scanner")
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			missingOSVScanner,
			"Binary for osv-scanner not found in PATH", "osv-scanner needs to be in your path.")
		return nil, nil
	}
	// exec
	cmdArgs := []string{"--json", "--lockfile", lockFile}
	data, err := exec.Command(path, cmdArgs...).Output()
	// error output is expected from osv-scanner
	if err != nil {
		if len(string(err.Error())) == 0 {
			pass.ReportResult(
				pass.AnalyzerName,
				scanningFailure,
				"error running osv-scanner",
				fmt.Sprintf("osv-scanner found, but failed to run: %s", err))
			return nil, nil
		}
	}

	// deserialize json output, detect CRITICAL severity

	var objmap OSVJsonOutput
	if err := json.Unmarshal(data, &objmap); err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			scanningFailure,
			"osv-scanner output not recognized",
			fmt.Sprintf("osv-scanner output could not be parsed: %s", err))
		return nil, nil
	}

	// iterate over results
	if len(objmap.Results) == 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			noIssues,
			"osv-scanner passed",
			fmt.Sprintf("osv-scanner detected no current issues for lockfile: %s", lockFile))
		return nil, nil
	}

	criticalSeverityCount := 0
	highSeverityCount := 0
	moderateSeverityCount := 0
	lowSeverityCount := 0

	for _, result := range objmap.Results {
		for _, aPackage := range result.Packages {
			for _, aVulnerability := range aPackage.Vulnerabilities {
				aliases := strings.Join(aVulnerability.Aliases, " ")
				message := fmt.Sprintf("SEVERITY: %s in package %s, vulnerable to %s", aVulnerability.DatabaseSpecific.Severity, aPackage.Package.Name, aliases)
				switch aVulnerability.DatabaseSpecific.Severity {
				case SeverityCritical:
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerCriticalSeverityDetected,
						"osv-scanner detected a critical severity issue",
						message)
					criticalSeverityCount++
				case SeverityHigh:
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerHighSeverityDetected,
						"osv-scanner detected a high severity issue",
						message)
					highSeverityCount++
				case SeverityModerate:
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerModerateSeverityDetected,
						"osv-scanner detected a moderate severity issue",
						message)
					moderateSeverityCount++
				case SeverityLow:
					pass.ReportResult(
						pass.AnalyzerName,
						osvScannerLowSeverityDetected,
						"osv-scanner detected a low severity issue",
						message)
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
			fmt.Sprintf("osv-scanner detected %d critical severity issues for lockfile: %s", criticalSeverityCount, lockFile))
	}
	if highSeverityCount > 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			osvScannerHighSeverityDetected,
			"osv-scanner detected high severity issues",
			fmt.Sprintf("osv-scanner detected %d high severity issues for lockfile: %s", highSeverityCount, lockFile))
	}
	if moderateSeverityCount > 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			osvScannerModerateSeverityDetected,
			"osv-scanner detected moderate severity issues",
			fmt.Sprintf("osv-scanner detected %d moderate severity issues for lockfile: %s", moderateSeverityCount, lockFile))
	}
	if lowSeverityCount > 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			osvScannerLowSeverityDetected,
			"osv-scanner detected low severity issues",
			fmt.Sprintf("osv-scanner detected %d low severity issues for lockfile: %s", lowSeverityCount, lockFile))
	}

	/*
		scanningSeverity.Severity = analysis.OK
		pass.ReportResult(
			pass.AnalyzerName,
			scanningSeverity,
			"osv-scanner passed",
			fmt.Sprintf("osv-scanner detected no current issues for lockfile: %s", lockFile))
	*/
	return nil, nil
}

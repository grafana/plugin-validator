package virusscan

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/scanprocess"
)

var (
	virusScanFailed = &analysis.Rule{
		Name:     "virus-scan-failed",
		Severity: analysis.Error,
	}
	virusScanPassed = &analysis.Rule{
		Name:     "virus-scan-passed",
		Severity: analysis.OK,
	}
)

type ClamAvScanSummary struct {
	ScannedDirs   int
	ScannedFiles  int
	InfectedFiles int
	KnownViruses  string
	EngineVersion string
	DataScanned   string
	DataRead      string
	ScanTime      string
	StartDate     string
	EndDate       string
	FoundFiles    []string
}

var Analyzer = &analysis.Analyzer{
	Name:     "virusscan",
	Requires: []*analysis.Analyzer{archive.Analyzer, sourcecode.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		virusScanFailed,
		virusScanPassed,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Virus Scan",
		Description:  "Runs a virus scan on the plugin archive and source code using `clamscan` (`clamav`).",
		Dependencies: "clamscan",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {

	skip := os.Getenv("SKIP_CLAMAV")
	if skip != "" {
		logme.Debugln("Skipping virus scan")
		return nil, nil
	}

	// check if clamav is installed
	clamavBin, err := exec.LookPath("clamscan")

	if err != nil {
		logme.Debugln("clamav not installed, skipping virus scan")
		return nil, nil
	}

	// scan the archive
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)

	if !ok {
		return nil, nil
	}

	logme.DebugFln("Will run clamav on %s", archiveDir)

	runClamavScan(clamavBin, archiveDir, "archive", pass)

	// scan the source code
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		// no source code found so we can't scan
		return nil, nil
	}

	logme.DebugFln("Will run clamav on %s", sourceCodeDir)

	runClamavScan(clamavBin, sourceCodeDir, "source code", pass)

	return nil, nil
}

func runClamavScan(clamavBin string, path string, entityName string, pass *analysis.Pass) {
	clamavCommand := exec.Command(clamavBin, "-r", path)
	clamavOutput, runErr := scanprocess.CombinedOutput(clamavCommand)
	scanSummary, parseErr := parseClamAv(string(clamavOutput))
	// Preserve positive findings even when ClamAV later fails or loses its summary.
	infectedFiles := max(scanSummary.InfectedFiles, len(scanSummary.FoundFiles))
	if infectedFiles > 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			virusScanFailed,
			fmt.Sprintf(
				"ClamAV found %d infected file(s) inside your %s",
				infectedFiles, entityName,
			),
			fmt.Sprintf("Files found by ClamAV: %s", strings.Join(scanSummary.FoundFiles, ", ")),
		)
	}

	// Exit 1 means malware was found. Report execution failures as diagnostics
	// and let the remaining scans run.
	var exitErr *exec.ExitError
	switch {
	case runErr != nil && (!errors.As(runErr, &exitErr) || exitErr.ExitCode() != 1):
		pass.ReportIncomplete(fmt.Sprintf("ClamAV %s scan failed: %v", entityName, runErr))
	case parseErr != nil:
		pass.ReportIncomplete(fmt.Sprintf("ClamAV %s scan output is incomplete: %v", entityName, parseErr))
	case exitErr != nil && infectedFiles == 0:
		pass.ReportIncomplete(fmt.Sprintf("ClamAV %s scan exited with findings but reported no infected files", entityName))
	default:
		if infectedFiles == 0 && virusScanPassed.ReportAll {
			pass.ReportResult(
				pass.AnalyzerName,
				virusScanPassed,
				"ClamAV found no infected files",
				"",
			)
		}
	}
}

func parseClamAv(output string) (ClamAvScanSummary, error) {
	scanSummary := ClamAvScanSummary{}

	scanner := bufio.NewScanner(strings.NewReader(output))
	summarySection := false
	foundInfectedCount := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.Contains(line, "SCAN SUMMARY") {
			summarySection = true
			continue
		}

		if summarySection {
			keyValue := strings.SplitN(line, ":", 2)
			if len(keyValue) != 2 {
				continue
			}
			key := strings.TrimSpace(keyValue[0])
			value := strings.TrimSpace(keyValue[1])

			switch key {
			case "Scanned directories":
				var err error
				scanSummary.ScannedDirs, err = strconv.Atoi(value)
				if err != nil {
					return scanSummary, fmt.Errorf("error parsing Scanned directories: %w", err)
				}
			case "Scanned files":
				var err error
				scanSummary.ScannedFiles, err = strconv.Atoi(value)
				if err != nil {
					return scanSummary, fmt.Errorf("error parsing Scanned files: %w", err)
				}
			case "Infected files":
				foundInfectedCount = true
				var err error
				scanSummary.InfectedFiles, err = strconv.Atoi(value)
				if err != nil {
					return scanSummary, fmt.Errorf("error parsing Infected files: %w", err)
				}
			case "Known viruses":
				scanSummary.KnownViruses = value
			case "Engine version":
				scanSummary.EngineVersion = value
			case "Data scanned":
				scanSummary.DataScanned = value
			case "Data read":
				scanSummary.DataRead = value
			case "Time":
				scanSummary.ScanTime = value
			case "Start Date":
				scanSummary.StartDate = value
			case "End Date":
				scanSummary.EndDate = value
			}
		} else {
			if strings.Contains(line, "FOUND") {
				fileStatus := strings.SplitN(line, ":", 2)
				if len(fileStatus) == 2 && strings.Contains(fileStatus[1], "FOUND") {
					scanSummary.FoundFiles = append(scanSummary.FoundFiles, strings.TrimSpace(fileStatus[0]))
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return scanSummary, fmt.Errorf("error scanning input: %w", err)
	}
	if !summarySection || !foundInfectedCount {
		return scanSummary, fmt.Errorf("missing scan summary or infected file count")
	}

	return scanSummary, nil
}

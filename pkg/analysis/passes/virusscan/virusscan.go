package virusscan

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
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
	archiveDir, ok := analysis.GetResult[string](pass, archive.Analyzer)

	if !ok {
		return nil, nil
	}

	logme.DebugFln("Will run clamav on %s", archiveDir)

	err = runClamavScan(clamavBin, archiveDir, "archive", pass)
	if err != nil {
		logme.Debugln("clamav failed, skipping virus scan", err)
		return nil, nil
	}

	// scan the source code
	sourceCodeDir, ok := analysis.GetResult[string](pass, sourcecode.Analyzer)
	if !ok {
		// no source code found so we can't scan
		return nil, nil
	}

	logme.DebugFln("Will run clamav on %s", archiveDir)

	err = runClamavScan(clamavBin, sourceCodeDir, "source code", pass)
	if err != nil {
		logme.Debugln("clamav failed, skipping virus scan", err)
		return nil, nil
	}

	return nil, nil
}

func runClamavScan(clamavBin string, path string, entityName string, pass *analysis.Pass) error {
	clamavCommand := exec.Command(clamavBin, "-r", path)
	clamavOutput, err := clamavCommand.CombinedOutput()

	// clamav exits 1 if it finds issues. Only failing if the output is empty
	if err != nil && len(clamavOutput) == 0 {
		logme.Debugln("clamav failed, skipping virus scan", err)
		return nil
	}

	scanSummary, err := parseClamAv(string(clamavOutput))
	if err != nil {
		logme.Debugln("error parsing clamav output, skipping virus scan", err)
		return nil
	}

	if scanSummary.InfectedFiles > 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			virusScanFailed,
			fmt.Sprintf(
				"ClamAV found %d infected file(s) inside your %s",
				scanSummary.InfectedFiles, entityName,
			),
			fmt.Sprintf("Files found by ClamAV: %s", strings.Join(scanSummary.FoundFiles, ", ")),
		)
	} else {
		if virusScanPassed.ReportAll {
			pass.ReportResult(
				pass.AnalyzerName,
				virusScanPassed,
				"ClamAV found no infected files",
				"",
			)
		}
	}
	return nil
}

func parseClamAv(output string) (ClamAvScanSummary, error) {
	scanSummary := ClamAvScanSummary{}

	scanner := bufio.NewScanner(strings.NewReader(output))
	summarySection := false

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

	return scanSummary, nil
}

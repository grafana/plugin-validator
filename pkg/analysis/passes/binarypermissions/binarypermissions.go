package binarypermissions

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	binaryExecutableFound       = &analysis.Rule{Name: "binary-executable-found", Severity: analysis.Error}
	binaryExecutablePermissions = &analysis.Rule{Name: "binary-executable-permissions", Severity: analysis.Error}
)

var REQUIRED_PERMISSIONS = fs.FileMode.Perm(0755)

var Analyzer = &analysis.Analyzer{
	Name:     "archivename",
	Requires: []*analysis.Analyzer{metadata.Analyzer, archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{binaryExecutableFound, binaryExecutablePermissions},
}

var binarySuffixes = []string{
	"_linux_amd64",
	"_linux_arm64",
	"_darwin_amd64",
	"_darwin_arm64",
	"_windows_amd64.exe",
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if data.Executable == "" && binaryExecutableFound.ReportAll {
		binaryExecutableFound.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, binaryExecutableFound, "No executable defined in plugin.json", "")
		return nil, nil
	}

	executable := data.Executable
	executableParentDir := filepath.Join(archiveDir, filepath.Dir(executable))
	executableName := filepath.Base(executable)

	var foundBinaries = []string{}
	for _, suffix := range binarySuffixes {
		binaryPath := filepath.Join(executableParentDir, executableName+suffix)
		if _, err := os.Stat(binaryPath); err == nil {
			continue
		}
		foundBinaries = append(foundBinaries, binaryPath)
	}

	if len(foundBinaries) == 0 {
		pass.ReportResult(pass.AnalyzerName,
			binaryExecutableFound,
			fmt.Sprintf("No binary found for `executable` %s defined in plugin.json", executable),
			fmt.Sprintf("You defined an executable %s but it could not be found for any of the supported architectures", executable))
		return nil, nil
	}

	for _, binaryPath := range foundBinaries {

		// skip windows executables
		if filepath.Ext(binaryPath) == ".exe" {
			continue
		}

		fileInfo, err := os.Stat(binaryPath)
		if err != nil {
			pass.ReportResult(pass.AnalyzerName,
				binaryExecutablePermissions,
				fmt.Sprintf("Could not read permissions for executable %s", binaryPath),
				fmt.Sprintf("Could not read the file %s. This could be too restrictive in your binary files or your zip file is corrupted", binaryPath))
		}

		filePermissions := fileInfo.Mode().Perm()
		if filePermissions != REQUIRED_PERMISSIONS {
			pass.ReportResult(pass.AnalyzerName,
				binaryExecutablePermissions,
				fmt.Sprintf("Permissions for binary executable %s are incorrect (%04o found).", filepath.Base(binaryPath), filePermissions),
				fmt.Sprintf("The binary file %s must have exact permissions %04o (%s).", filepath.Base(binaryPath), REQUIRED_PERMISSIONS, REQUIRED_PERMISSIONS))
		}
	}

	return nil, nil
}

package binarypermissions

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
)

var (
	binaryExecutableFound = &analysis.Rule{
		Name:     "binary-executable-found",
		Severity: analysis.Error,
	}
	binaryExecutablePermissions = &analysis.Rule{
		Name:     "binary-executable-permissions",
		Severity: analysis.Error,
	}
	archiveFilesError = &analysis.Rule{
		Name:     "archive-files-error",
		Severity: analysis.Error,
	}
)

var requiredPermissions = fs.FileMode.Perm(0755)

var Analyzer = &analysis.Analyzer{
	Name:     "binarypermissions",
	Requires: []*analysis.Analyzer{archive.Analyzer, nestedmetadata.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		binaryExecutableFound,
		binaryExecutablePermissions,
		archiveFilesError,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	metadatamap, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		return nil, nil
	}

	for pluginJsonPath, data := range metadatamap {
		if data.Executable == "" {
			if binaryExecutableFound.ReportAll {
				binaryExecutableFound.Severity = analysis.OK
				pass.ReportResult(
					pass.AnalyzerName,
					binaryExecutableFound,
					"No executable defined in plugin.json",
					fmt.Sprintf("no executable defined in plugin.json: %s", pluginJsonPath),
				)
			}
			continue
		}
		relativeTo := filepath.Join(archiveDir, filepath.Dir(pluginJsonPath))
		executable := data.Executable
		executableParentDir := filepath.Join(relativeTo, filepath.Dir(executable))
		executableName := filepath.Base(executable)

		// walk all files in executableParentDir
		var foundBinaries = []string{}
		err := filepath.WalkDir(
			executableParentDir,
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if d.Type().IsRegular() {
					if strings.HasPrefix(d.Name(), executableName) {
						foundBinaries = append(foundBinaries, path)
					}
				}
				return nil
			},
		)
		if err != nil {
			pass.ReportResult(
				pass.AnalyzerName,
				archiveFilesError,
				"error reading your archive files",
				fmt.Sprintf("error reading your archive files: %s", err.Error()),
			)
			continue
		}
		if len(foundBinaries) == 0 {
			pass.ReportResult(
				pass.AnalyzerName,
				binaryExecutableFound,
				fmt.Sprintf(
					"No binary found for `executable` %s defined in %s", executable, pluginJsonPath,
				),
				fmt.Sprintf(
					"You defined an executable %s but it could not be found for any of the supported architectures",
					executable,
				),
			)
			continue
		}

		for _, binaryPath := range foundBinaries {

			// skip windows executables
			if filepath.Ext(binaryPath) == ".exe" {
				continue
			}

			fileInfo, err := os.Stat(binaryPath)
			if err != nil {
				pass.ReportResult(
					pass.AnalyzerName,
					binaryExecutablePermissions,
					fmt.Sprintf("Could not read permissions for executable %s", binaryPath),
					fmt.Sprintf(
						"Could not read the file %s. This could be too few permissions in your binary files or your zip file is corrupted",
						binaryPath,
					),
				)
				continue
			}

			filePermissions := fileInfo.Mode().Perm()
			if filePermissions != requiredPermissions {
				pass.ReportResult(
					pass.AnalyzerName,
					binaryExecutablePermissions,
					fmt.Sprintf(
						"Permissions for binary executable %s are incorrect (%04o found).",
						filepath.Base(binaryPath),
						filePermissions,
					),
					fmt.Sprintf(
						"The binary file %s must have exact permissions %04o (%s).",
						filepath.Base(binaryPath),
						requiredPermissions,
						requiredPermissions,
					),
				)
			}
		}
	}

	return nil, nil
}

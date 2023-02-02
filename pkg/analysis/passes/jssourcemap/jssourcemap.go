package jssourcemap

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/difftool"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	jsMapNotFound = &analysis.Rule{Name: "js-map-not-found", Severity: analysis.Error}
	jsMapInvalid  = &analysis.Rule{Name: "js-map-invalid", Severity: analysis.Error}
	jsMapNoMatch  = &analysis.Rule{Name: "js-map-no-match", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "jsMap",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer, archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{jsMapNotFound},
}

func run(pass *analysis.Pass) (interface{}, error) {

	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok || sourceCodeDir == "" {
		return nil, nil
	}

	archiveFilesPath, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok || archiveFilesPath == "" {
		return nil, nil
	}

	archiveModuleJs, err := findModuleJsFiles(archiveFilesPath)
	if err != nil {
		return nil, err
	}

	mapFiles := []string{}
	for _, file := range archiveModuleJs {
		fileMapPath := filepath.Join(filepath.Dir(file), "module.js.map")
		_, err := os.Stat(fileMapPath)
		if err != nil {
			if os.IsNotExist(err) {
				fileMapRelPath, _ := filepath.Rel(archiveFilesPath, fileMapPath)
				pass.ReportResult(pass.AnalyzerName, jsMapNotFound, fmt.Sprintf("missing %s in archive", fileMapRelPath), "You must include generated source maps for your plugin in your archive file. If you have nested plugins, you must include the source maps for each plugin")
			} else {
				return nil, err
			}
		} else {
			mapFiles = append(mapFiles, fileMapPath)
		}
	}

	// do not continue if not all map files were found
	if len(mapFiles) != len(archiveModuleJs) {
		return nil, nil
	}

	sourceCodeDirSrc := filepath.Join(sourceCodeDir, "src")
	if err != nil {
		return nil, err
	}
	for _, file := range mapFiles {
		diffReport, err := difftool.CompareSourceMapToSourceCode(file, sourceCodeDirSrc)
		if err != nil {
			fmt.Println("found error")
			pass.ReportResult(pass.AnalyzerName, jsMapInvalid, fmt.Sprintf("the sourcemap file %s could not be validated", file), "You must include generated source maps for your plugin in your archive file. If you have nested plugins, you must include the source maps for each plugin")
			logme.DebugFln("could not extract source map: %s", err)
			return nil, nil
		}

		if diffReport.TotalDifferences != 0 {
			pass.ReportResult(pass.AnalyzerName, jsMapNoMatch, "The provided javascript/typescript source code does not match your plugin archive assets.", "Verify the provided source code is the same as the one used to generate plugin archive. If you are providing a git repository URL make sure to include the correct ref (branch or tag) in the URL")
			return nil, nil
		}
	}

	return nil, nil
}

func findModuleJsFiles(archivePath string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(archivePath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "module.js") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

package jssourcemap

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/difftool"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/utils"
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
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "JS Source Map",
		Description:  "Checks for required `module.js.map` file(s) in archive.",
		Dependencies: "`sourceCodeUri`",
	},
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

	archiveModuleJs, err := doublestar.FilepathGlob(archiveFilesPath + "/**/module.js")

	if err != nil {
		return nil, err
	}

	var mapFiles []string
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

	// get the plugin id from the archive
	pluginID, err := utils.GetPluginId(archiveFilesPath)
	if err != nil || pluginID == "" {
		pass.ReportResult(pass.AnalyzerName, jsMapInvalid, "plugin.json not found in src", "Required plugin.json must exist in src")
		return nil, nil
	}

	for _, file := range mapFiles {
		diffReport, err := difftool.CompareSourceMapToSourceCode(pluginID, file, sourceCodeDirSrc)
		if err != nil {
			pass.ReportResult(pass.AnalyzerName, jsMapInvalid, fmt.Sprintf("the sourcemap file %s could not be validated", file), "You must include generated source maps for your plugin in your archive file. If you have nested plugins, you must include the source maps for each plugin")
			logme.DebugFln("could not extract source map: %s", err)
			return nil, nil
		}

		if diffReport.TotalDifferences != 0 {
			reportText := diffReport.GeneratePrintableReport()
			logme.DebugFln("source map mismatch for %s", file)
			logme.Debugln("---REPORT START--")
			logme.Debugln("\n\n" + reportText)
			logme.Debugln("---REPORT-END---")

			pass.ReportResult(pass.AnalyzerName, jsMapNoMatch, "The provided javascript/typescript source code does not match your plugin archive assets.", "Verify the provided source code is the same as the one used to generate plugin archive. If you are providing a git repository URL make sure to include the correct ref (branch or tag) in the URL. \n "+reportText)
			return nil, nil
		} else {
			logme.DebugFln("source map matches source code")
		}
	}

	return nil, nil
}

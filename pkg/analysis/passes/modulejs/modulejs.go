package modulejs

import (
	"os"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	missingModulejs = &analysis.Rule{Name: "missing-modulejs", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "modulejs",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingModulejs},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok || archiveDir == "" {
		// this should never happen
		return nil, nil
	}

	//find all module.js files with doublestar
	moduleJsFiles, err := doublestar.FilepathGlob(archiveDir + "/**/module.js")
	if err != nil {
		return nil, nil
	}

	if len(moduleJsFiles) == 0 {
		pass.ReportResult(pass.AnalyzerName, missingModulejs, "missing module.js", "Your plugin must have a module.js file to be loaded by Grafana.")
		return nil, nil
	} else if missingModulejs.ReportAll {
		missingModulejs.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, missingModulejs, "module.js: exists", "")
	}

	moduleJsFilesContent := map[string][]byte{}

	for _, moduleJsFile := range moduleJsFiles {
		content, err := os.ReadFile(moduleJsFile)
		if err != nil {
			return nil, err
		}
		moduleJsFilesContent[moduleJsFile] = content
	}

	return moduleJsFilesContent, nil
}

package jssourcemap

import (
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
)

var (
	jsMapNotFound = &analysis.Rule{Name: "js-map-not-found", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "jsMap",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer, archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{jsMapNotFound},
}

func run(pass *analysis.Pass) (interface{}, error) {

	// if no sourceCode provided skip this analyzer
	if pass.ResultOf[sourcecode.Analyzer] == nil {
		return nil, nil
	}
	sourceCodeDir := pass.ResultOf[sourcecode.Analyzer].(string)
	if sourceCodeDir == "" {
		return nil, nil
	}

	archiveFilesPath := pass.ResultOf[archive.Analyzer].(string)

	archiveJsMaps, err := filepath.Glob(filepath.Join(archiveFilesPath, "**/module.js.map"))
	if err != nil {
		return nil, err
	}

	if len(archiveJsMaps) == 0 {
		pass.ReportResult(pass.AnalyzerName, jsMapNotFound, "no module.js.map found in archive", "You must include generated source maps for your plugin in your archive file.")
	}

	return nil, nil
}

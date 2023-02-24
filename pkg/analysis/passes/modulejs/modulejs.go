package modulejs

import (
	"os"
	"path/filepath"

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
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	b, err := os.ReadFile(filepath.Join(archiveDir, "module.js"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.ReportResult(pass.AnalyzerName, missingModulejs, "missing module.js", "Your plugin must have a module.js file to be loaded by Grafana.")
			return nil, nil
		}
		return nil, err
	} else {
		if missingModulejs.ReportAll {
			missingModulejs.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, missingModulejs, "module.js: exists", "")
		}
	}

	return b, nil
}

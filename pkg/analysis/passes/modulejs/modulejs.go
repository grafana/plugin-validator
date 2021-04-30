package modulejs

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	missingModulejs = &analysis.Rule{Name: "missing-modulejs"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "modulejs",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingModulejs},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	b, err := ioutil.ReadFile(filepath.Join(archiveDir, "module.js"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.Reportf(pass.AnalyzerName, missingModulejs, "missing module.js")
			return nil, nil
		}
		return nil, err
	} else {
		if missingModulejs.ReportAll {
			missingModulejs.Severity = analysis.OK
			pass.Reportf(pass.AnalyzerName, missingModulejs, "module.js: exists")
		}
	}

	return b, nil
}

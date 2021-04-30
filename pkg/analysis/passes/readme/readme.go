package readme

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	missingReadme = &analysis.Rule{Name: "missing-readme"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "readme",
	Run:      run,
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Rules:    []*analysis.Rule{missingReadme},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	b, err := ioutil.ReadFile(filepath.Join(archiveDir, "README.md"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.Reportf(pass.AnalyzerName, missingReadme, "missing README.md")
			return nil, nil
		}
		return nil, err
	} else {
		if missingReadme.ReportAll {
			missingReadme.Severity = analysis.OK
			pass.Reportf(pass.AnalyzerName, missingReadme, "README.md: exists")
		}
	}

	return b, nil
}

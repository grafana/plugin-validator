package readme

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var Analyzer = &analysis.Analyzer{
	Name:     "readme",
	Run:      run,
	Requires: []*analysis.Analyzer{archive.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	b, err := ioutil.ReadFile(filepath.Join(archiveDir, "README.md"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.Report(analysis.Diagnostic{
				Severity: analysis.Error,
				Message:  "missing README.md",
			})
			return nil, nil
		}
		return nil, err
	}

	return b, nil
}

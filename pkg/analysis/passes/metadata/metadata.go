package metadata

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var Analyzer = &analysis.Analyzer{
	Name:     "metadata",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	b, err := ioutil.ReadFile(filepath.Join(archiveDir, "plugin.json"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.Report(analysis.Diagnostic{
				Severity: analysis.Error,
				Message:  "missing plugin.json",
			})
			return nil, nil
		}
		return nil, err
	}

	return b, nil
}

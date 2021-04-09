package manifest

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	unsignedPlugin = &analysis.Rule{Name: "unsigned-plugin"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "manifest",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{unsignedPlugin},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	b, err := ioutil.ReadFile(filepath.Join(archiveDir, "MANIFEST.txt"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.Reportf(unsignedPlugin, "unsigned plugin")
			return nil, nil
		}
		return nil, err
	}

	return b, nil
}

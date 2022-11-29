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
			pass.Reportf(pass.AnalyzerName, unsignedPlugin, "unsigned plugin", "MANIFEST.txt file not found. Please refer to the documentation for how to sign a plugin.")
			return nil, nil
		} else {
			if unsignedPlugin.ReportAll {
				unsignedPlugin.Severity = analysis.OK
				pass.Reportf(pass.AnalyzerName, unsignedPlugin, "MANIFEST.txt: plugin is signed")
			}
		}
		return nil, err
	}

	return b, nil
}

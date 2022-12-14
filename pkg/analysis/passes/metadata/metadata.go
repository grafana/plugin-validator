package metadata

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	missingMetadata = &analysis.Rule{Name: "missing-metadata"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "metadata",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingMetadata},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	b, err := ioutil.ReadFile(filepath.Join(archiveDir, "plugin.json"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.ReportResult(pass.AnalyzerName, missingMetadata, "missing plugin.json", "A plugin.json file is required to describe the plugin.")
			return nil, nil
		} else {
			if missingMetadata.ReportAll {
				missingMetadata.Severity = analysis.OK
				pass.ReportResult(pass.AnalyzerName, missingMetadata, "plugin.json exists", "")
			}
		}
		return nil, err
	}

	return b, nil
}

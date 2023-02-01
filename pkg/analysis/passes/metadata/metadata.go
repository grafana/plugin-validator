package metadata

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	missingMetadata = &analysis.Rule{Name: "missing-metadata", Severity: analysis.Error}
)

// TODO: static analyzers: remove

var Analyzer = &analysis.Analyzer{Name: "metadata"}

type StaticMetadataAnalyzer struct {
	analysis.Analyzer

	Result []byte
}

func NewStaticAnalyzer() *StaticMetadataAnalyzer {
	return &StaticMetadataAnalyzer{
		Analyzer: analysis.NewAnalyzer("metadata").
			WithDependencies(archive.Analyzer.Name).
			WithRules(missingMetadata),
	}
}

// TODO: static analyzers: remove

func (a *StaticMetadataAnalyzer) GetResult() interface{} {
	return a.Result
}

func (a *StaticMetadataAnalyzer) Run(pass *analysis.Pass) error {
	archiveDir := pass.DependencyResults.Archive
	b, err := ioutil.ReadFile(filepath.Join(archiveDir, "plugin.json"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.ReportResult(pass.AnalyzerName, missingMetadata, "missing plugin.json", "A plugin.json file is required to describe the plugin.")
			return nil
		}
		if missingMetadata.ReportAll {
			missingMetadata.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, missingMetadata, "plugin.json exists", "")
		}
		return err
	}
	a.Result = b
	return nil
}

var _ = analysis.StaticAnalyzer(&StaticMetadataAnalyzer{})

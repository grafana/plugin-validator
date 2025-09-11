package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	missingMetadata = &analysis.Rule{Name: "missing-metadata", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "metadata",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingMetadata},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Metadata",
		Description: "Checks that `plugin.json` exists and is valid.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)

	if !ok {
		return nil, nil
	}

	b, err := os.ReadFile(filepath.Join(archiveDir, "plugin.json"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.ReportResult(
				pass.AnalyzerName,
				missingMetadata,
				"missing plugin.json",
				"A plugin.json file is required to describe the plugin. Please see https://grafana.com/developers/plugin-tools/publish-a-plugin/package-a-plugin for more information on how to package a plugin.",
			)
			return nil, nil
		}
		return nil, err
	}
	var data Metadata
	if err := json.Unmarshal(b, &data); err != nil {
		// if we fail to unmarshall it means the schema is incorrect
		// we will let the metadatavaid validator handle it
		return nil, nil
	}

	return b, nil
}

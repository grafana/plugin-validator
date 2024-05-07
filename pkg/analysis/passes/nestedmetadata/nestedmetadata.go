package nestedmetadata

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	missingMetadata      = &analysis.Rule{Name: "missing-metadata", Severity: analysis.Error}
	errorReadingMetadata = &analysis.Rule{Name: "error-reading-metadata", Severity: analysis.Error}
	invalidMetadata      = &analysis.Rule{Name: "invalid-metadata", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "nestedmetadata",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingMetadata, errorReadingMetadata},
}

type Metadatamap map[string]metadata.Metadata

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)

	if !ok {
		return nil, nil
	}

	pluginJsonPath, err := doublestar.FilepathGlob(archiveDir + "/**/plugin.json")
	if err != nil {

		pass.ReportResult(
			pass.AnalyzerName,
			errorReadingMetadata,
			"error reading plugin.json",
			"Error reading plugin.json in your plugin archive: "+err.Error(),
		)
		return nil, nil
	}

	// this is technically the same as the metadata analyzer check
	// the logic is kept here duplicated to eventually remove the metadata
	// analyzer and only keep this one
	if len(pluginJsonPath) == 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			missingMetadata,
			"missing plugin.json",
			"A plugin.json file is required to describe the plugin. No plugin.json was found in your plugin archive.",
		)
	}

	mainPluginJsonFile := filepath.Join(archiveDir, "plugin.json")

	pluginJsonFiles := make(Metadatamap)

	for _, path := range pluginJsonPath {
		metadataBody, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				pass.ReportResult(
					pass.AnalyzerName,
					missingMetadata,
					"missing plugin.json",
					"A plugin.json file is required to describe the plugin. No plugin.json was found in your plugin archive.",
				)
				return nil, nil
			} else {
				if missingMetadata.ReportAll {
					missingMetadata.Severity = analysis.OK
					pass.ReportResult(pass.AnalyzerName, missingMetadata, "plugin.json exists", "")
				}
			}
			return nil, err
		}

		var data metadata.Metadata
		if err := json.Unmarshal(metadataBody, &data); err != nil {
			pass.ReportResult(
				pass.AnalyzerName,
				invalidMetadata,
				"Invalid plugin.json in your archive.",
				"The plugin.json file is not valid and can't be parsed. Please refer to the documentation for more information. https://grafana.com/developers/plugin-tools/reference/plugin-json",
			)
			return nil, err
		}
		if path == mainPluginJsonFile {
			pluginJsonFiles["plugin.json"] = data
		} else {
			pluginJsonFiles[path] = data
		}
	}

	return pluginJsonFiles, nil
}

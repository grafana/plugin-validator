package metadatavalid

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadataschema"
)

var (
	invalidMetadata  = &analysis.Rule{Name: "invalid-metadata", Severity: analysis.Error}
	metadataNotFound = &analysis.Rule{Name: "metadata-not-found", Severity: analysis.Error}
	validMetadata    = &analysis.Rule{Name: "valid-metadata", Severity: analysis.OK}
)

var Analyzer = &analysis.Analyzer{
	Name:     "metadatavalid",
	Requires: []*analysis.Analyzer{metadata.Analyzer, metadataschema.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{invalidMetadata, metadataNotFound, validMetadata},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Metadata Validity",
		Description: "Ensures metadata is valid and matches plugin schema.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	schema, ok := analysis.GetResult[[]byte](pass, metadataschema.Analyzer)
	if !ok {
		return nil, nil
	}

	schemaFile, err := os.CreateTemp("", "plugin_*.schema.json")
	if err != nil {
		return nil, fmt.Errorf("couldn't create schema file: %w", err)
	}
	defer os.Remove(schemaFile.Name())

	_, err = io.Copy(schemaFile, bytes.NewReader(schema))
	if err != nil {
		return nil, fmt.Errorf("couldn't create schema file: %w", err)
	}

	// gojsonschema requires absolute path to the schema.
	schemaPath, err := filepath.Abs(schemaFile.Name())
	if err != nil {
		return nil, err
	}

	archiveDir, ok := analysis.GetResult[string](pass, archive.Analyzer)
	if !ok {
		return nil, nil
	}

	// Using the path here rather than the result of metadata.Analyzer since
	// gojsonschema needs an actual file.
	// we don't use the result of metadata.Analyzer because that validator can fail
	// if the metadata is incorrect
	metadataPath, err := filepath.Abs(filepath.Join(archiveDir, "plugin.json"))
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(metadataPath)
	switch {
	case os.IsNotExist(err):
		pass.ReportResult(
			pass.AnalyzerName,
			metadataNotFound,
			"plugin.json not found",
			"plugin.json not found in the archive. Please refer to the documentation for more information. https://grafana.com/docs/grafana/latest/developers/plugins/metadata/",
		)
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("%q stat: %w", metadataPath, err)
	case err == nil:
		break
	}
	schemaLoader := gojsonschema.NewReferenceLoader("file:///" + schemaPath)
	documentLoader := gojsonschema.NewReferenceLoader("file:///" + metadataPath)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, err
	}

	for _, desc := range result.Errors() {
		pass.ReportResult(
			pass.AnalyzerName,
			invalidMetadata,
			fmt.Sprintf("plugin.json: %s: %s", desc.Field(), desc.Description()),
			"The plugin.json file is not following the schema. Please refer to the documentation for more information. https://grafana.com/docs/grafana/latest/developers/plugins/metadata/",
		)
	}
	if len(result.Errors()) == 0 && validMetadata.ReportAll {
		pass.ReportResult(pass.AnalyzerName, validMetadata, "plugin.json: metadata is valid", "")
	}

	return nil, nil
}

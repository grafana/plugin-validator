package metadatavalid

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadataschema"
	"github.com/xeipuuv/gojsonschema"
)

var (
	invalidMetadata = &analysis.Rule{Name: "invalid-metadata", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "metadatavalid",
	Requires: []*analysis.Analyzer{metadata.Analyzer, metadataschema.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{invalidMetadata},
}

func run(pass *analysis.Pass) (interface{}, error) {
	schema := pass.ResultOf[metadataschema.Analyzer].([]byte)

	schemaFile, err := os.TempFile("", "plugin_*.schema.json")
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

	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	// Using the path here rather than the result of metadata.Analyzer since
	// gojsonschema needs an actual file.
	metadataPath, err := filepath.Abs(filepath.Join(archiveDir, "plugin.json"))
	if err != nil {
		return nil, err
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file:///" + schemaPath)
	documentLoader := gojsonschema.NewReferenceLoader("file:///" + metadataPath)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, err
	}

	for _, desc := range result.Errors() {
		pass.ReportResult(pass.AnalyzerName, invalidMetadata, fmt.Sprintf("plugin.json: %s: %s", desc.Field(), desc.Description()), "The plugin.json file is not following the schema. Please refer to the documentation for more information. https://grafana.com/docs/grafana/latest/developers/plugins/metadata/")
	}
	if len(result.Errors()) == 0 && invalidMetadata.ReportAll {
		invalidMetadata.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, invalidMetadata, "plugin.json: metadata is valid", "")
	}

	return nil, nil
}

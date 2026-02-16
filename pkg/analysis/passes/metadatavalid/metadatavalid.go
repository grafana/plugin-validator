package metadatavalid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/xeipuuv/gojsonschema"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadataschema"
	"github.com/grafana/plugin-validator/pkg/logme"
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
	schema, ok := pass.ResultOf[metadataschema.Analyzer].([]byte)
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

	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
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

	pluginJsonBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		// log the error and continue with schema validation
		logme.ErrorF("failed to read plugin.json in metadatavalid check: %q", err)
	}
	if pluginJsonBytes != nil {
		var data metadata.Metadata
		if err := json.Unmarshal(pluginJsonBytes, &data); err == nil {
			_, err = version.NewConstraint(data.Dependencies.GrafanaDependency)
			if err != nil {
				pass.ReportResult(
					pass.AnalyzerName,
					invalidMetadata,
					fmt.Sprintf("plugin.json: Dependencies.grafanaDependency field has invalid or empty version constraint: %q", data.Dependencies.GrafanaDependency),
					"The plugin.json file is not following the schema. Please refer to the documentation for more information. https://grafana.com/docs/grafana/latest/developers/plugins/metadata/#grafanadependency",
				)
			}
		}
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file:///" + schemaPath)
	documentLoader := gojsonschema.NewReferenceLoader("file:///" + metadataPath)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, err
	}

	errLen := len(result.Errors())
	for _, desc := range result.Errors() {
		// we validate grafanaDependency at line 91-100,
		// so we ignore the error from schema validation
		if strings.Contains(desc.Field(), "grafanaDependency") || strings.Contains(desc.Description(), "grafanaDependency") {
			errLen -= 1
			continue
		}
		pass.ReportResult(
			pass.AnalyzerName,
			invalidMetadata,
			fmt.Sprintf("plugin.json: %s: %s", desc.Field(), desc.Description()),
			"The plugin.json file is not following the schema. Please refer to the documentation for more information. https://grafana.com/docs/grafana/latest/developers/plugins/metadata/",
		)
	}
	if errLen == 0 && validMetadata.ReportAll {
		pass.ReportResult(pass.AnalyzerName, validMetadata, "plugin.json: metadata is valid", "")
	}

	return nil, nil
}

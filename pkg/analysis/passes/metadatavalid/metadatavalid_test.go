package metadatavalid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadataschema"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

var schemaContent []byte

func TestMetadataValid(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:        filepath.Join("testdata", "valid"),
			metadataschema.Analyzer: getSchema(),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestMetadataInvalid(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:        filepath.Join("testdata", "invalid"),
			metadataschema.Analyzer: getSchema(),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"plugin.json: (root): Additional property invalid is not allowed",
		interceptor.Diagnostics[0].Title,
	)
}

func getSchema() []byte {
	if len(schemaContent) > 0 {
		return schemaContent
	}
	schemaPath := filepath.Join("testdata", "schema.json")
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		panic(err)

	}
	return schemaContent
}

package templatereadme

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestTemplateReadme(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	readmeContent := []byte(`# Grafana Panel Plugin Template`)
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: readmeContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "README.md: uses README from template")
}

func TestTemplateReadmeLowerCase(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	readmeContent := []byte(`# Grafana panel Plugin Template`)
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: readmeContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "README.md: uses README from template")
}

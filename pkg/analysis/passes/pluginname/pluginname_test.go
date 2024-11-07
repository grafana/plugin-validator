package pluginname

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestValidPluginName(t *testing.T) {
	const pluginId = "raintank-plugin-panel"
	var interceptor testpassinterceptor.TestPassInterceptor

	publishedStatus := &published.PluginStatus{
		Status:  "active",
		Version: "1.0.0",
		Slug:    pluginId,
	}

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:  []byte(`{"ID": "` + pluginId + `", "name": "my plugin name", "info": {"logos": {"large": "img/logo.svg"}}}`),
			published.Analyzer: publishedStatus,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestInvalidPluginName(t *testing.T) {
	const pluginId = "raintank-plugin-panel"
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer:  []byte(`{"ID": "` + pluginId + `", "name": "` + pluginId + `", "info": {"logos": {"large": "img/logo.svg"}}}`),
			published.Analyzer: nil,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "plugin.json: plugin name should be human-friendly")
}

package grafanadependency

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestGrafanaDependency(t *testing.T) {
	for _, tc := range []struct {
		name       string
		pluginJSON string
		titleMsg   string
	}{
		{
			name: "valid grafanaDependency constraint",
			pluginJSON: `{
				"id": "test-org-app",
				"dependencies": { "grafanaDependency": ">=11.6.0" }
			}`,
			titleMsg: "",
		},
		{
			name: "complex but valid grafanaDependency constraint",
			pluginJSON: `{
				"id": "test-org-app",
				"dependencies": { "grafanaDependency": ">=11.6.11 <12 || >=12.0.10 <12.1 || >=12.1.7 <12.2 || >=12.2.5" }
			}`,
			titleMsg: "",
		},
		{
			name: "invalid grafanaDependency constraint",
			pluginJSON: `{
				"id": "test-org-app",
				"dependencies": { "grafanaDependency": ">=invalid" }
			}`,
			titleMsg: "plugin.json: dependencies.grafanaDependency field has invalid or empty version constraint: \">=invalid\"",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{
				RootDir: filepath.Join("./"),
				ResultOf: map[*analysis.Analyzer]interface{}{
					metadata.Analyzer: []byte(tc.pluginJSON),
				},
				Report: interceptor.ReportInterceptor(),
			}

			_, err := Analyzer.Run(pass)
			require.NoError(t, err)
			if len(tc.titleMsg) > 0 {
				require.Len(t, interceptor.Diagnostics, 1)
				require.Equal(
					t,
					tc.titleMsg,
					interceptor.Diagnostics[0].Title,
				)
			} else {
				require.Len(t, interceptor.Diagnostics, 0)
			}
		})
	}
}

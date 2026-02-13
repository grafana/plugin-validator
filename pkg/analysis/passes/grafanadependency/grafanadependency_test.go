package grafanadependency

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadatavalid"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestGrafanaDependency(t *testing.T) {
	for _, tc := range []struct {
		name       string
		pluginJSON string
		expFlag    bool
	}{
		{
			name: "non-grafana labs plugin without pre-release shouldn't be flagged",
			pluginJSON: `{
				"id": "community-my-app",
				"dependencies": { "grafanaDependency": ">=11.6.0" }
			}`,
			expFlag: false,
		},
		{
			name: "non-grafana labs plugin with pre-release shouldn't be flagged",
			pluginJSON: `{
				"id": "community-my-app",
				"dependencies": { "grafanaDependency": ">=11.6.0-0" }
			}`,
			expFlag: false,
		},
		{
			name: "grafana org plugin without pre-release should be flagged",
			pluginJSON: `{
				"id": "grafana-my-app",
				"dependencies": { "grafanaDependency": ">=11.6.0" }
			}`,
			expFlag: true,
		},
		{
			name: "grafana labs author plugin without pre-release should be flagged",
			pluginJSON: `{
				"id": "adopted-my-app",
				"info": { "author": { "name": "Grafana Labs" } },
				"dependencies": { "grafanaDependency": ">=11.6.0" }
			}`,
			expFlag: true,
		},
		{
			name: "grafana org plugin with pre-release should not be flagged",
			pluginJSON: `{
				"id": "grafana-my-app",
				"dependencies": { "grafanaDependency": ">= 11.6.0-0" }
			}`,
			expFlag: false,
		},
		{
			name: "grafana labs author plugin with pre-release should not be flagged",
			pluginJSON: `{
				"id": "adopted-my-app",
				"info": { "author": { "name": "Grafana Labs" } },
				"dependencies": { "grafanaDependency": ">= 11.6.0-0" }
			}`,
			expFlag: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{
				RootDir: filepath.Join("./"),
				ResultOf: map[*analysis.Analyzer]interface{}{
					metadata.Analyzer:      []byte(tc.pluginJSON),
					archive.Analyzer:       filepath.Join("."),
					metadatavalid.Analyzer: nil,
				},
				Report: interceptor.ReportInterceptor(),
			}

			_, err := Analyzer.Run(pass)
			require.NoError(t, err)
			if tc.expFlag {
				require.Len(t, interceptor.Diagnostics, 1)

				var meta metadata.Metadata
				require.NoError(t, json.Unmarshal(pass.ResultOf[metadata.Analyzer].([]byte), &meta))
				grafanaDependency := meta.Dependencies.GrafanaDependency
				require.NotEmpty(t, grafanaDependency, "grafana dependency should not be empty")

				d := interceptor.Diagnostics[0]
				require.Equal(t, analysis.Warning, d.Severity, "severity should be warning")
				require.Equal(t, "missing-cloud-pre-release", d.Name, "name should match")
				require.Equal(t, fmt.Sprintf(`Grafana dependency "%s" has no pre-release value`, grafanaDependency), d.Title, "title should match")
				require.Equal(t, fmt.Sprintf(`The value of grafanaDependency in plugin.json ("%s") is missing a pre-release value. This may make the plugin uninstallable in Grafana Cloud. Please add "-0" as a suffix of your grafanaDependency value ("%s-0")`, grafanaDependency, grafanaDependency), d.Detail, "detail should match")
			} else {
				ok := assert.Len(t, interceptor.Diagnostics, 0, "expecting no diagnostics but got %d", len(interceptor.Diagnostics))
				// Log for debugging
				if !ok {
					for _, d := range interceptor.Diagnostics {
						t.Logf("%+v", d)
					}
				}
			}
		})
	}
}

func TestGrafanaDependency_GetPreRelease(t *testing.T) {
	for _, tc := range []struct {
		name       string
		dependency string
		expPre     string
	}{
		{"no pre-release", ">=12.4.0", ""},
		{"no pre-release with space", ">= 12.4.0", ""},
		{"zero pre-release", ">=12.4.0-0", "0"},
		{"zero pre-release with space", ">= 12.4.0-0", "0"},
		{"non-zero pre-release", ">=12.4.0-1189998819991197253", "1189998819991197253"},
		{"non-zero pre-release with space", ">= 12.4.0-1189998819991197253", "1189998819991197253"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pre := getPreRelease(tc.dependency)
			require.Equal(t, tc.expPre, pre, "extracted pre-release value should match")
		})
	}
}

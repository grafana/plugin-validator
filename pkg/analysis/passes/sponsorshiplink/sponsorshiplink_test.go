package sponsorshiplink

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestValidSponsorshipLink(t *testing.T) {
	testCases := []struct {
		name     string
		linkName string
		url      string
	}{
		{"sponsorship as name", "sponsorship", "https://example.com/sponsorMe"},
		{"sponsor as name", "Sponsor", "https://example.com/becomeSponsor"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var interceptor testpassinterceptor.TestPassInterceptor
			pluginJsonContent := `{
				"name": "my plugin name",
				"info": {
				"links": [
				{
					"url": "https://example.com/website",
					"name": "Website"
				},	
				{
					"url": "` + tc.url + `",
					"name": "` + tc.linkName + `"
				}
			]
			}
			}`
			pass := &analysis.Pass{
				RootDir: filepath.Join("./"),
				ResultOf: map[*analysis.Analyzer]interface{}{
					metadata.Analyzer: []byte(pluginJsonContent),
					archive.Analyzer:  filepath.Join("."),
				},
				Report: interceptor.ReportInterceptor(),
			}

			_, err := Analyzer.Run(pass)
			require.NoError(t, err)
			require.Len(t, interceptor.Diagnostics, 0)
		})
	}
}

func TestNoSponsorshipLink(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
		"name": "my plugin name",
		"info": {
		"links": [
			{
				"url": "https://example.com/documentation",
				"name": "Documentation"
			}
		]
		}
	}`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("."),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Severity, analysis.Recommendation)
	require.Equal(t, interceptor.Diagnostics[0].Title, "You can include a sponsorship link if you want users to support your work")
}

func TestEmptyLinks(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	const pluginJsonContent = `{
		"name": "my plugin name",
		"info": {
		"links": []
		}
	}`
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(pluginJsonContent),
			archive.Analyzer:  filepath.Join("."),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Severity, analysis.Recommendation)
	require.Equal(t, interceptor.Diagnostics[0].Title, "You can include a sponsorship link if you want users to support your work")
}

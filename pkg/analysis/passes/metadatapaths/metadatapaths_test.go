package metadatapaths

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/logos"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/screenshots"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

type analysisSetterFunc func(m map[*analysis.Analyzer]interface{}, k *analysis.Analyzer) error

func unmarshalJSONOfType(typ interface{}, payload string) analysisSetterFunc {
	return func(m map[*analysis.Analyzer]interface{}, k *analysis.Analyzer) error {
		rv := reflect.New(reflect.TypeOf(typ))
		iface := rv.Interface()
		if err := json.Unmarshal([]byte(payload), &iface); err != nil {
			return err
		}
		m[k] = rv.Elem().Interface()
		return nil
	}
}

func TestMetadatapaths(t *testing.T) {
	type tc struct {
		name     string
		resultOf map[*analysis.Analyzer]interface{}
		exp      []string
	}

	for _, tc := range []tc{
		{
			name: "with correct metadata",
			resultOf: map[*analysis.Analyzer]interface{}{
				logos.Analyzer: unmarshalJSONOfType(
					metadata.MetadataLogos{},
					`{"small": "img/logo.svg", "large": "img/logo.svg"}`,
				),
				screenshots.Analyzer: unmarshalJSONOfType(
					[]metadata.MetadataScreenshots{},
					`[{"path": "img/screenshot.png", "name": "test"}]`,
				),
			},
			exp: nil,
		},
		{
			name: "with wrong logo path",
			resultOf: map[*analysis.Analyzer]interface{}{
				logos.Analyzer: unmarshalJSONOfType(
					metadata.MetadataLogos{},
					`{"small": "/img/wrong-with-slash.svg", "large": "./img/wrong-with-dot.svg"}`,
				),
				screenshots.Analyzer: unmarshalJSONOfType(
					[]metadata.MetadataScreenshots{},
					`[{"name": "test", "path": "img/screenshots.png"}]`,
				),
			},
			exp: []string{
				"plugin.json: relative small logo path should not start with",
				"plugin.json: relative large logo path should not start with",
			},
		},
		{
			name: "with wrong screenshots path",
			resultOf: map[*analysis.Analyzer]interface{}{
				logos.Analyzer: unmarshalJSONOfType(
					metadata.MetadataLogos{},
					`{"small": "img/logo.svg", "large": "img/logo.svg"}`,
				),
				screenshots.Analyzer: unmarshalJSONOfType(
					[]metadata.MetadataScreenshots{},
					`[{"name": "test", "path": "/img/wrong-with-slash.png"}, {"name": "test2", "path":"./img/wrong-with-dot"}]`,
				),
			},
			exp: []string{
				"plugin.json: relative screenshot path should not start with",
				"plugin.json: relative screenshot path should not start with",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resultOf := make(map[*analysis.Analyzer]interface{}, len(tc.resultOf))
			for k, v := range tc.resultOf {
				if f, ok := v.(analysisSetterFunc); ok {
					// Call the analysisSetterFunc
					require.NoError(t, f(resultOf, k))
				} else {
					// Plain value
					resultOf[k] = v
				}
			}

			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{
				RootDir:  filepath.Join("./"),
				ResultOf: resultOf,
				Report:   interceptor.ReportInterceptor(),
			}
			_, err := Analyzer.Run(pass)
			require.NoError(t, err)

			// Check exp by len first
			assert.Len(t, interceptor.Diagnostics, len(tc.exp), "wrong number of diagnostics compared to expectations")

			// Check content of all expectations
			for _, d := range interceptor.Diagnostics {
				for i, e := range tc.exp {
					if strings.Contains(d.Title, e) {
						// Exp met, delete it
						tc.exp[i] = tc.exp[len(tc.exp)-1]
						tc.exp = tc.exp[:len(tc.exp)-1]
						break
					}
				}
			}

			// All exp met <=> all exps have been deleted <=> empty slice
			assert.Empty(t, tc.exp, "some expectations haven't been met")
		})
	}
}

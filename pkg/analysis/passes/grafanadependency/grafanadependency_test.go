package grafanadependency

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
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
				"dependencies": { "grafanaDependency": ">=invalid2" }
			}`,
			titleMsg: "plugin.json: dependencies.grafanaDependency field has invalid or empty version constraint: \">=invalid2\"",
		},
		{
			name: "empty grafanaDependency",
			pluginJSON: `{
				"id": "test-org-app",
				"dependencies": { "grafanaDependency": "" }
			}`,
			titleMsg: "plugin.json: dependencies.grafanaDependency field has invalid or empty version constraint: \"\"",
		},
		{
			name: "missing grafanaDependency",
			pluginJSON: `{
				"id": "test-org-app",
				"dependencies": {"plugins": [ ]}
			}`,
			titleMsg: "plugin.json: dependencies.grafanaDependency field has invalid or empty version constraint: \"\"",
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

func TestNestedGrafanaDependency(t *testing.T) {
	const parent = ">12.1.1"
	const child = "12.3.2"
	const compound = ">=11.6.11 <12 || >=12.0.10 <12.1 || >=12.1.7 <12.2 || >=12.2.5"

	childPath := filepath.Join("datasource", "plugin.json")
	deepPath := filepath.Join("datasource", "panel", "plugin.json")
	matchingPath := filepath.Join("matching", "plugin.json")
	dependencyJSON := func(constraint string) string {
		return fmt.Sprintf(`{"dependencies":{"grafanaDependency":%q}}`, constraint)
	}

	reportAll := validGrafanaDependency.ReportAll
	validGrafanaDependency.ReportAll = true
	t.Cleanup(func() { validGrafanaDependency.ReportAll = reportAll })

	for _, tc := range []struct {
		name        string
		parentJSON  string
		nested      map[string]string
		omitNested  bool
		mismatches  []string
		invalidRoot bool
		invalidJSON bool
	}{
		{
			name:   "matching constraints",
			nested: map[string]string{childPath: dependencyJSON(parent)},
		},
		{
			name:       "matching compound constraints",
			parentJSON: dependencyJSON(compound),
			nested:     map[string]string{childPath: dependencyJSON(compound)},
		},
		{
			name:       "child version satisfies parent but differs",
			nested:     map[string]string{childPath: dependencyJSON(child)},
			mismatches: []string{childPath},
		},
		{
			name:       "equivalent constraints must match exactly",
			nested:     map[string]string{childPath: dependencyJSON("> 12.1.1")},
			mismatches: []string{childPath},
		},
		{
			name: "all mismatches in path order including deeper nesting",
			nested: map[string]string{
				matchingPath: dependencyJSON(parent),
				deepPath:     dependencyJSON(child),
				childPath:    dependencyJSON(child),
			},
			mismatches: []string{deepPath, childPath},
		},
		{
			name:   "no nested plugins and root entry is skipped",
			nested: map[string]string{nestedmetadata.MainPluginJson: dependencyJSON(child)},
		},
		{
			name:   "empty discovery result",
			nested: map[string]string{},
		},
		{
			name:       "absent discovery result",
			omitNested: true,
		},
		{
			name: "typed nil discovery result",
		},
		{
			name:       "empty child constraint",
			nested:     map[string]string{childPath: dependencyJSON("")},
			mismatches: []string{childPath},
		},
		{
			name:       "missing child constraint",
			nested:     map[string]string{childPath: `{"dependencies":{}}`},
			mismatches: []string{childPath},
		},
		{
			name:       "invalid child constraint is only a mismatch",
			nested:     map[string]string{childPath: dependencyJSON("invalid")},
			mismatches: []string{childPath},
		},
		{
			name:        "invalid parent retains root validation",
			parentJSON:  dependencyJSON(">=invalid2"),
			nested:      map[string]string{childPath: dependencyJSON(child)},
			invalidRoot: true,
		},
		{
			name:        "empty parent retains root validation",
			parentJSON:  dependencyJSON(""),
			nested:      map[string]string{childPath: dependencyJSON(child)},
			invalidRoot: true,
		},
		{
			name:        "missing parent retains root validation",
			parentJSON:  `{"dependencies":{}}`,
			nested:      map[string]string{childPath: dependencyJSON(child)},
			invalidRoot: true,
		},
		{
			name:        "malformed parent retains unmarshal behavior",
			parentJSON:  `{"dependencies":`,
			nested:      map[string]string{childPath: dependencyJSON(child)},
			invalidJSON: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parentJSON := tc.parentJSON
			if parentJSON == "" {
				parentJSON = dependencyJSON(parent)
			}

			var nested nestedmetadata.Metadatamap
			if tc.nested != nil {
				nested = make(nestedmetadata.Metadatamap)
				for path, raw := range tc.nested {
					var data metadata.Metadata
					require.NoError(t, json.Unmarshal([]byte(raw), &data))
					nested[path] = data
				}
			}

			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{
				AnalyzerName: Analyzer.Name,
				ResultOf: map[*analysis.Analyzer]interface{}{
					metadata.Analyzer: []byte(parentJSON),
				},
				Report: interceptor.ReportInterceptor(),
			}
			if !tc.omitNested {
				pass.ResultOf[nestedmetadata.Analyzer] = nested
			}

			_, err := Analyzer.Run(pass)
			require.NoError(t, err)
			if tc.invalidJSON {
				require.Empty(t, interceptor.Diagnostics)
			} else if tc.invalidRoot {
				require.Len(t, interceptor.Diagnostics, 1)
				require.Equal(t, "invalid-grafana-dependency", interceptor.Diagnostics[0].Name)
				require.Equal(t, analysis.Error, interceptor.Diagnostics[0].Severity)
			} else if len(tc.mismatches) > 0 {
				require.Len(t, interceptor.Diagnostics, len(tc.mismatches))
				for i, path := range tc.mismatches {
					assertMismatch(t, interceptor.Diagnostics[i], path, parent, nested[path].Dependencies.GrafanaDependency)
				}
			} else {
				require.Len(t, interceptor.Diagnostics, 1)
				require.Equal(t, "valid-grafana-dependency", interceptor.Diagnostics[0].Name)
				require.Equal(t, analysis.OK, interceptor.Diagnostics[0].Severity)
			}
		})
	}
}

func TestNestedDependencyPipeline(t *testing.T) {
	require.Contains(t, Analyzer.Requires, metadata.Analyzer)
	require.Contains(t, Analyzer.Requires, nestedmetadata.Analyzer)

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "nested"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "nested"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	for _, analyzer := range []*analysis.Analyzer{metadata.Analyzer, nestedmetadata.Analyzer} {
		pass.AnalyzerName = analyzer.Name
		result, err := analyzer.Run(pass)
		require.NoError(t, err)
		pass.ResultOf[analyzer] = result
	}
	require.Empty(t, interceptor.Diagnostics)

	pass.AnalyzerName = Analyzer.Name
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	assertMismatch(t, interceptor.Diagnostics[0], filepath.Join("datasource", "panel", "plugin.json"), ">12.1.1", "12.3.2")
	assertMismatch(t, interceptor.Diagnostics[1], filepath.Join("datasource", "plugin.json"), ">12.1.1", "12.3.2")
}

func assertMismatch(t *testing.T, diagnostic *analysis.Diagnostic, path, parent, child string) {
	t.Helper()
	require.Equal(t, "mismatched-grafana-dependency", diagnostic.Name)
	require.Equal(t, analysis.Error, diagnostic.Severity)
	require.Equal(t, fmt.Sprintf("%s: dependencies.grafanaDependency %q does not match root plugin.json constraint %q", path, child, parent), diagnostic.Title)
}

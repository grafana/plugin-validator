package backenddebug

import (
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strings"
	"testing"
)

var (
	pluginJSONWithExecutable    = []byte(`{"executable": "gpx_plugin"}`)
	pluginJSONWithoutExecutable = []byte(`{}`)
)

func TestBackendDebug_Correct(t *testing.T) {
	for _, tc := range []struct {
		name       string
		folder     string
		pluginJSON []byte
	}{
		{
			name:       "with executable",
			folder:     "correct",
			pluginJSON: pluginJSONWithExecutable,
		},
		{
			name:       "standalone-txt without executable",
			folder:     "standalone-txt",
			pluginJSON: pluginJSONWithoutExecutable,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{
				RootDir: filepath.Join("testdata", tc.folder),
				ResultOf: map[*analysis.Analyzer]interface{}{
					archive.Analyzer:  filepath.Join("testdata", tc.folder),
					metadata.Analyzer: tc.pluginJSON,
				},
				Report: interceptor.ReportInterceptor(),
			}
			_, err := Analyzer.Run(pass)
			require.NoError(t, err)
			require.Empty(t, interceptor.Diagnostics)
		})
	}
}

func TestBackendDebug(t *testing.T) {
	for _, tc := range []struct {
		name            string
		folder          string
		pluginJSON      []byte
		failureFileName []string
	}{
		{
			name:            "standalone-txt",
			folder:          "standalone-txt",
			pluginJSON:      pluginJSONWithExecutable,
			failureFileName: []string{"standalone.txt"},
		},
		{
			name:            "pid-txt",
			folder:          "pid-txt",
			pluginJSON:      pluginJSONWithExecutable,
			failureFileName: []string{"pid.txt"},
		},
		{
			name:            "all",
			folder:          "all",
			pluginJSON:      pluginJSONWithExecutable,
			failureFileName: []string{"standalone.txt", "pid.txt"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{
				RootDir: filepath.Join("testdata", tc.folder),
				ResultOf: map[*analysis.Analyzer]interface{}{
					archive.Analyzer:  filepath.Join("testdata", tc.folder),
					metadata.Analyzer: tc.pluginJSON,
				},
				Report: interceptor.ReportInterceptor(),
			}
			_, err := Analyzer.Run(pass)
			require.NoError(t, err)

			// Expect error
			require.Len(t, interceptor.Diagnostics, len(tc.failureFileName))
			found := map[string]struct{}{}
			for _, d := range interceptor.Diagnostics {
				require.Equal(t, analysis.Error, d.Severity)
				require.Equal(t, "backend-debug-file-present", d.Name)
				require.Equal(t, "found standalone backend file", d.Title)
				for _, expFn := range tc.failureFileName {
					if !strings.Contains(d.Detail, expFn) {
						continue
					}
					if _, ok := found[expFn]; ok {
						require.Failf(t, "found %q twice", expFn)
						return
					}
					found[expFn] = struct{}{}
				}
			}
			require.Len(t, found, len(tc.failureFileName))
		})
	}
}

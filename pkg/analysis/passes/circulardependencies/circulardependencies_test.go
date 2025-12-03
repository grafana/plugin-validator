package circulardependencies

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/grafana/plugin-validator/pkg/testutils"
)

func TestCircularDependencies(t *testing.T) {
	t.Run("itself", func(t *testing.T) {
		pass, interceptor := newTestPass(filepath.Join("testdata", "itself"))
		require.NoError(t, testutils.RunDependencies(pass, Analyzer))

		_, err := Analyzer.Run(pass)
		require.NoError(t, err)
		require.Len(t, interceptor.Diagnostics, 1)
		require.Equal(t, "Circular dependency detected: grafana-clock-panel -> grafana-clock-panel", interceptor.Diagnostics[0].Title)
	})

	t.Run("with a nested plugin", func(t *testing.T) {
		pass, interceptor := newTestPass(filepath.Join("testdata", "nested"))
		require.NoError(t, testutils.RunDependencies(pass, Analyzer))

		_, err := Analyzer.Run(pass)
		require.NoError(t, err)
		checkCircularDependencies(
			t,
			interceptor.Diagnostics,
			"grafana-clock-panel -> grafana-nested-panel",
			"grafana-nested-panel -> grafana-clock-panel",
		)
	})

	const (
		gcomLatestVersionURL = "https://grafana.com/api/plugins/grafana-external-panel/versions/latest"
	)

	t.Run("with an external plugin", func(t *testing.T) {
		t.Run("with external dependency", func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			httpmock.RegisterResponder(http.MethodGet, gcomLatestVersionURL, httpmock.NewStringResponder(http.StatusOK, gcomAPIResponse))

			pass, interceptor := newTestPass(filepath.Join("testdata", "external-with-version"))
			require.NoError(t, testutils.RunDependencies(pass, Analyzer))

			_, err := Analyzer.Run(pass)
			require.NoError(t, err)
			checkCircularDependencies(
				t,
				interceptor.Diagnostics,
				"grafana-clock-panel -> grafana-external-panel",
			)
			info := httpmock.GetCallCountInfo()
			require.Equal(t, 1, info["GET "+gcomLatestVersionURL])
		})

		t.Run("without version field", func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			httpmock.RegisterResponder(http.MethodGet, gcomLatestVersionURL, httpmock.NewStringResponder(http.StatusOK, gcomAPIResponse))

			pass, interceptor := newTestPass(filepath.Join("testdata", "external-without-version"))
			require.NoError(t, testutils.RunDependencies(pass, Analyzer))

			_, err := Analyzer.Run(pass)
			require.NoError(t, err)
			checkCircularDependencies(
				t,
				interceptor.Diagnostics,
				"grafana-clock-panel -> grafana-external-panel",
			)
			info := httpmock.GetCallCountInfo()
			require.Equal(t, 1, info["GET "+gcomLatestVersionURL])
		})
	})

	t.Run("gcom error should not return an error", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder(
			http.MethodGet,
			gcomLatestVersionURL,
			httpmock.NewStringResponder(http.StatusInternalServerError, "not a json response"),
		)

		pass, interceptor := newTestPass(filepath.Join("testdata", "external-without-version"))
		require.NoError(t, testutils.RunDependencies(pass, Analyzer))
		_, err := Analyzer.Run(pass)
		require.NoError(t, err)
		require.Len(t, interceptor.Diagnostics, 0)
		info := httpmock.GetCallCountInfo()
		require.Equal(t, 1, info["GET "+gcomLatestVersionURL])
	})
}

func newTestPass(rootDir string) (*analysis.Pass, *testpassinterceptor.TestPassInterceptor) {
	var interceptor testpassinterceptor.TestPassInterceptor
	return &analysis.Pass{
		RootDir:  rootDir,
		ResultOf: map[*analysis.Analyzer]interface{}{},
		Report:   interceptor.ReportInterceptor(),
	}, &interceptor
}

func checkCircularDependencies(t *testing.T, gotDiagnostics []*analysis.Diagnostic, expSubStrings ...string) {
	exp := make(map[string]struct{}, len(expSubStrings))
	for _, e := range expSubStrings {
		exp[e] = struct{}{}
	}
	for _, d := range gotDiagnostics {
		for e := range exp {
			if !strings.Contains(d.Title, e) {
				continue
			}
			delete(exp, e)
		}
	}
	require.Emptyf(t, exp, "all expected dependencies should be found. Couldn't find: %+v", exp)
}

const gcomAPIResponse = `{
  "json": {
    "dependencies": {
      "grafanaDependency": "\u003e=8.0.0",
      "plugins": [
        {
          "id": "grafana-clock-panel",
          "name": "Clock",
          "type": "panel"
        }
      ]
    },
    "id": "grafana-external-panel",
    "info": {
      "author": {
        "name": "Grafana Labs",
        "url": "https://grafana.com"
      },
      "build": {
        "time": 1672825174542,
        "repo": "https://github.com/grafana/clock-panel",
        "branch": "master",
        "hash": "5a0962f98d3ff043bad0db32dc804999a669b161",
        "build": 57
      },
      "description": "Clock panel for grafana",
      "keywords": [
        "clock",
        "panel"
      ],
      "links": [
        {
          "name": "Project site",
          "url": "https://github.com/grafana/clock-panel"
        },
        {
          "name": "MIT License",
          "url": "https://github.com/grafana/clock-panel/blob/master/LICENSE"
        }
      ],
      "logos": {
        "large": "img/clock.svg",
        "small": "img/clock.svg"
      },
      "screenshots": [
        {
          "name": "Showcase",
          "path": "img/screenshot-showcase.png"
        },
        {
          "name": "Options",
          "path": "img/screenshot-clock-options.png"
        }
      ],
      "updated": "2023-01-04",
      "version": "2.1.2"
    },
    "name": "External",
    "skipDataQuery": true,
    "type": "panel"
  }
}`

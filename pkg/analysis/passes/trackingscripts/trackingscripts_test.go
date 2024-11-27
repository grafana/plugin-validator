package trackingscripts

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestTrackingScriptsValid(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	var moduleJsMap = map[string][]byte{
		"module.js": []byte(`import { PanelPlugin } from '@grafana/data'`),
	}
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			modulejs.Analyzer: moduleJsMap,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestTrackingScriptsInvalid(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	var moduleJsMap = map[string][]byte{
		"module.js": []byte(`https://www.google-analytics.com/analytics.js`),
	}
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			modulejs.Analyzer: moduleJsMap,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Title,
		"module.js: should not include tracking scripts",
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Detail,
		"Tracking scripts are not allowed in Grafana plugins (e.g. google analytics). Please remove any usage of tracking code. Found: https://www.google-analytics.com/analytics.js",
	)
}

func TestNoFalsePositiveForSringsLookingLikeDomains(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	var moduleJsMap = map[string][]byte{
		"module.js": []byte(`grafana-asserts-app.rules:read`),
	}
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			modulejs.Analyzer: moduleJsMap,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

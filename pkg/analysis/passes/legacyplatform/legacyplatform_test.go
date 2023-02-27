package legacyplatform

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

const pluginId = "test-plugin-panel" //nolint:golint,unused

func TestLegacyPlatformUsesCurrentPlatform(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			modulejs.Analyzer: &map[string][]byte{"module.js": []byte(`import { PanelPlugin } from '@grafana/data'`)},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

var legacyImportTests = []map[string][]byte{
	{"module.js": []byte(`import { MetricsPanelCtrl } from 'grafana/app/plugins/sdk';`)},
	{"module.js": []byte(`define(["app/plugins/sdk"],(function(n){return function(n){var t={};function e(r){if(t[r])return t[r].exports;var o=t[r]={i:r,l:!1,exports:{}};return n[r].call(o.exports,o,o.exports,e),o.l=!0,o.exports}return e.m=n,e.c=t,e.d=function(n,t,r){e.o(n,t)||Object.defineProperty(n,t,{enumerable:!0,get:r})},e.r=function(n){"undefined"!=typeof`)},
	{"module.js": []byte(`define(["app/plugins/sdk"],(function(n){return function(n){var t={};function e(r){if(t[r])return t[r].exports;var o=t[r]={i:r,l:!1,exports:{}};return n[r].call(o.exports,o,o.exports,e),o.l=!0,o.exports}return e.m=n,e.c=t,e.d=function(n,t,r){e.o(n,t)||Object.defineProperty(n,t,{enumerable:!0,get:r})},e.r=function(n){"undefined"!=typeof Symbol&&Symbol.toSt`)},
	{"module.js": []byte(`define(["react","lodash","@grafana/data","@grafana/ui","@emotion/css","@grafana/runtime","moment","app/core/utils/datemath","jquery","app/plugins/sdk","app/core/core_module","app/core/core","app/core/table_model","app/core/utils/kbn","app/core/config","angular"],(function(e,t,r,n,i,a,o,s,u,l,c,p,f,h,d,m){return function(e){var t={};function r(n){if(t[n])return t[n].exports;var i=t[n]={i:n,l:!1,exports:{}};retur`)},
}

func TestLegacyPlatformUsesLegacy(t *testing.T) {

	for _, moduleJsMap := range legacyImportTests {
		var interceptor testpassinterceptor.TestPassInterceptor

		pass := &analysis.Pass{
			RootDir: filepath.Join("./"),
			ResultOf: map[*analysis.Analyzer]interface{}{
				modulejs.Analyzer: &moduleJsMap,
			},
			Report: interceptor.ReportInterceptor(),
		}

		_, err := Analyzer.Run(pass)
		require.NoError(t, err)
		require.Len(t, interceptor.Diagnostics, 1)
		require.Equal(t, interceptor.Diagnostics[0].Title, "module.js: uses legacy plugin platform")
		require.Equal(t, interceptor.Diagnostics[0].Severity, analysis.Error)
	}
}

func TestOnlyWarnInPublishedPlugins(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginStatus := published.PluginStatus{
		Status:  "active",
		Slug:    pluginId,
		Version: "1.0.0",
	}

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			modulejs.Analyzer:  &map[string][]byte{"module.js": []byte(`import { MetricsPanelCtrl } from 'grafana/app/plugins/sdk';`)},
			published.Analyzer: &pluginStatus,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "module.js: uses legacy plugin platform")
	require.Equal(t, interceptor.Diagnostics[0].Severity, analysis.Warning)
}

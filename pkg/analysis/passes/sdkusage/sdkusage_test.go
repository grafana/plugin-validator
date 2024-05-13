package sdkusage

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/grafana/plugin-validator/pkg/utils"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func TestGoModNotFound(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`)

	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "nogomod"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"go.mod can not be found in your source code",
		interceptor.Diagnostics[0].Title,
	)
}

func TestGoModNotParseable(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`)
	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "gomodwrong"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"go.mod can not be parsed from your source code",
		interceptor.Diagnostics[0].Title,
	)
}

func TestValidGoMod(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// mock latest request
	httpmock.RegisterResponder(
		"GET",
		"https://api.github.com/repos/grafana/grafana-plugin-sdk-go/releases/latest",
		httpmock.NewStringResponder(
			200,
			`{ "tag_name": "v0.230.0", "published_at": "2024-05-09T10:03:16Z" }`,
		),
	)

	// mock tag request
	httpmock.RegisterResponder(
		"GET",
		"https://api.github.com/repos/grafana/grafana-plugin-sdk-go/releases/tags/v0.225.0",
		httpmock.NewStringResponder(
			200,
			`{ "tag_name": "v0.225.0", "published_at": "2024-04-18T11:07:23Z" }`,
		),
	)

	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`)
	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "validgomod"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestValidGoModWithNoGrafanaSdk(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`)
	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "nografanagosdk"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Your plugin uses a backend (backend=true), but the Grafana go sdk is not used",
		interceptor.Diagnostics[0].Title,
	)
}

func TestTwoMonthsOldSdk(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// mock latest request
	httpmock.RegisterResponder(
		"GET",
		"https://api.github.com/repos/grafana/grafana-plugin-sdk-go/releases/latest",
		httpmock.NewStringResponder(
			200,
			`{ "tag_name": "v0.230.0", "published_at": "2024-05-09T10:03:16Z" }`,
		),
	)

	// mock tag request
	httpmock.RegisterResponder(
		"GET",
		"https://api.github.com/repos/grafana/grafana-plugin-sdk-go/releases/tags/v0.212.0",
		httpmock.NewStringResponder(
			200,
			`{ "tag_name": "v0.212.0", "published_at": "2024-02-19T13:06:21Z" }`,
		),
	)

	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`)
	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "sdk-2-months-old"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)

	require.Equal(
		t,
		"Your Grafana go sdk is older than 2 months",
		interceptor.Diagnostics[0].Title,
	)
}

func TestFiveMonthsOld(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// mock latest request
	httpmock.RegisterResponder(
		"GET",
		"https://api.github.com/repos/grafana/grafana-plugin-sdk-go/releases/latest",
		httpmock.NewStringResponder(
			200,
			`{ "tag_name": "v0.230.0", "published_at": "2024-05-09T10:03:16Z" }`,
		),
	)

	// mock tag request
	httpmock.RegisterResponder(
		"GET",
		"https://api.github.com/repos/grafana/grafana-plugin-sdk-go/releases/tags/v0.187.0",
		httpmock.NewStringResponder(
			200,
			`{ "tag_name": "v0.187.0", "published_at": "2023-10-19T11:06:45Z" }`,
		),
	)

	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
    "name": "my plugin name",
    "backend": true,
    "executable": "gx_plugin"
  }`)
	meta, err := utils.JSONToMetadata(pluginJsonContent)
	require.NoError(t, err)

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "sdk-5-months-old"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				"plugin.json": meta,
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)

	require.Equal(
		t,
		"Your Grafana go sdk is older than 5 months",
		interceptor.Diagnostics[0].Title,
	)
}

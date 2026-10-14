package requiredlinks

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func runWithLinks(t *testing.T, links string) []string {
	t.Helper()

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(`{"info": {"links": ` + links + `}}`),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	var titles []string
	for _, d := range interceptor.Diagnostics {
		require.Equal(t, analysis.Error, d.Severity)
		titles = append(titles, d.Title)
	}
	return titles
}

func TestRuleIsDisabledByDefault(t *testing.T) {
	require.True(t, missingRequiredLink.DisabledByDefault)
}

func TestAllLinksPresent(t *testing.T) {
	titles := runWithLinks(t, `[
		{"name": "License", "url": "https://example.com/license"},
		{"name": "Privacy Notice", "url": "https://example.com/privacy"},
		{"name": "Customer SUPPORT", "url": "https://example.com/support"}
	]`)

	require.Empty(t, titles)
}

func TestNoLinks(t *testing.T) {
	titles := runWithLinks(t, `[]`)

	require.Equal(t, []string{
		"plugin.json: missing License link",
		"plugin.json: missing Privacy Policy link",
		"plugin.json: missing Support link",
	}, titles)
}

func TestInvalidURLs(t *testing.T) {
	titles := runWithLinks(t, `[
		{"name": "License", "url": "not a url"},
		{"name": "Privacy Policy", "url": "ftp://example.com/privacy"},
		{"name": "Support", "url": "https://"}
	]`)

	require.Equal(t, []string{
		"plugin.json: License link has an invalid URL",
		"plugin.json: Privacy Policy link has an invalid URL",
		"plugin.json: Support link has an invalid URL",
	}, titles)
}

func TestHTTPURLIsInvalid(t *testing.T) {
	titles := runWithLinks(t, `[
		{"name": "License", "url": "http://example.com/license"},
		{"name": "Privacy Policy", "url": "https://example.com/privacy"},
		{"name": "Support", "url": "https://example.com/support"}
	]`)

	require.Equal(t, []string{"plugin.json: License link has an invalid URL"}, titles)
}

func TestOneValidMatchIsEnough(t *testing.T) {
	titles := runWithLinks(t, `[
		{"name": "License", "url": ""},
		{"name": "License (full text)", "url": "https://example.com/license"},
		{"name": "Privacy Policy", "url": "https://example.com/privacy"},
		{"name": "Support", "url": "https://example.com/support"}
	]`)

	require.Empty(t, titles)
}

package manifest

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestWithNoManifest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "no-manifest"),
			published.Analyzer: &published.PluginStatus{
				Status: "published",
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "unsigned plugin", interceptor.Diagnostics[0].Title)
	require.Equal(t, "MANIFEST.txt file not found. Please refer to the documentation for how to sign a plugin. https://grafana.com/docs/grafana/latest/developers/plugins/sign-a-plugin/", interceptor.Diagnostics[0].Detail)
}

func TestWithNoManifestNewPlugin(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "no-manifest"),
			published.Analyzer: &published.PluginStatus{
				Status: "unknown",
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "unsigned plugin", interceptor.Diagnostics[0].Title)
	require.Equal(t, "This is a new (unpublished) plugin. This is expected during the initial review process. Please allow the review to continue, and a member of our team will inform you when your plugin can be signed.", interceptor.Diagnostics[0].Detail)
}

func TestWithEmptyManfiest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "with-empty-manifest"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "empty manifest")
}

func TestManifestWithAllFiles(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "with-all-files"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestManifestWithMissingFiles(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "manifest-missing-files"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(t, interceptor.Diagnostics[0].Title, "undeclared files in MANIFEST")

	messages := []string{
		"File img/clock-extra.svg is not declared in MANIFEST.txt",
		"File not-declared.js is not declared in MANIFEST.txt",
	}
	require.ElementsMatch(t, messages, interceptor.GetDetails())
}

func TestManifestWithExtraFiles(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "manifest-extra-files"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	require.Equal(t, interceptor.Diagnostics[0].Title, "declared files in MANIFEST not present")

	messages := []string{
		"File extra-file.js is declared in MANIFEST.txt but does not exist",
		"File img/extra-file.js is declared in MANIFEST.txt but does not exist",
	}
	require.ElementsMatch(t, messages, interceptor.GetDetails())
}

func TestBadFormedManifest(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "bad-formed-manifest"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "could not parse MANIFEST.txt")

}

func TestInvalidChecksum(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "with-wrong-sha-sum"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "invalid file checksum")
	require.Equal(t, interceptor.Diagnostics[0].Detail, "checksum for file module.js is invalid")
}

func TestCommunityWithRootUrls(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "with-root-urls"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "MANIFEST.txt: plugin signature contains rootUrls")
	require.Equal(t, interceptor.Diagnostics[0].Detail, "The plugin is signed as community but contains rootUrls. Do not pass --rootUrls when signing this plugin as community type")
}

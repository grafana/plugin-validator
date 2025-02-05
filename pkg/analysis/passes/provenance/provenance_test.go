package provenance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func canRunProvenanceTest() bool {
	return os.Getenv("GITHUB_TOKEN") != ""
}

func TestNoGithubUrlForAsset(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	if !canRunProvenanceTest() {
		t.Skip("github token not set")
	}

	pass := &analysis.Pass{
		RootDir:  filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{},
		Report:   interceptor.ReportInterceptor(),
		CheckParams: analysis.CheckParams{
			SourceCodeReference: "https://static.grafana.com/plugin.zip",
		},
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	// should skip the validation for non github urls
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"No provenance attestation. This plugin was built without build verification",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		"Cannot verify plugin build. It is recommended to use a pipeline that supports provenance attestation, such as GitHub Actions. https://github.com/grafana/plugin-actions/tree/main/build-plugin",
		interceptor.Diagnostics[0].Detail,
	)

}

func TestValidBuildProvenanceAttestion(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	if !canRunProvenanceTest() {
		t.Skip("github token not set")
	}

	pass := &analysis.Pass{
		RootDir:  filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{},
		Report:   interceptor.ReportInterceptor(),
		CheckParams: analysis.CheckParams{
			SourceCodeReference: "https://github.com/grafana/provenance-test-plugin/releases/download/v1.0.7/grafana-provenancetest-panel-1.0.7.zip",
			ArchiveFile: filepath.Join(
				"testdata",
				"valid",
				"grafana-provenancetest-panel-1.0.7.zip",
			),
		},
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestInvalidBuildProvenanceAttestion(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	if !canRunProvenanceTest() {
		t.Skip("github token not set")
	}

	pass := &analysis.Pass{
		RootDir:  filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{},
		Report:   interceptor.ReportInterceptor(),
		CheckParams: analysis.CheckParams{
			SourceCodeReference: "https://github.com/grafana/provenance-test-plugin/releases/download/v1.0.7/grafana-provenancetest-panel-1.0.8.zip",
			ArchiveFile: filepath.Join(
				"testdata",
				"invalid",
				"grafana-provenancetest-panel-1.0.8.zip",
			),
		},
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Cannot verify plugin build provenance attestation.",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		"Please verify your workflow attestation settings. See the documentation on implementing build attestation: https://github.com/grafana/plugin-actions/tree/main/build-plugin#add-attestation-to-your-existing-workflow",
		interceptor.Diagnostics[0].Detail,
	)
}

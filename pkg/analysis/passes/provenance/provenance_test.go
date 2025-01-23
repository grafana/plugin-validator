package provenance

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func canRunProvenanceTest() bool {
	githubCliBin, err := getGithubCliPath()
	return err == nil && githubCliBin != ""
}

func TestNoGithubUrlForAsset(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	if !canRunProvenanceTest() {
		t.Skip("github cli not installed")
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
		interceptor.Diagnostics[0].Title,
		"No provenance attestation. This plugin was built without build verification",
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Detail,
		"Cannot verify plugin build. It is recommended to use a pipeline that supports provenance attestation, such as GitHub Actions. https://github.com/grafana/plugin-actions/tree/main/build-plugin",
	)

}

func TestValidBuildProvenanceAttestion(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	if !canRunProvenanceTest() {
		t.Skip("github cli not installed")
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
		t.Skip("github cli not installed")
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
		interceptor.Diagnostics[0].Title,
		"Cannot verify plugin build. Attestation not found for asset testdata/invalid/grafana-provenancetest-panel-1.0.8.zip",
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Detail,
		"Please verify your workflow attestation settings. See the documentation on implementing build attestation: https://github.com/grafana/plugin-actions/tree/main/build-plugin#add-attestation-to-your-existing-workflow",
	)
}

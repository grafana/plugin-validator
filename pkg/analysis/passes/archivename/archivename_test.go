package archivename

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/assert"
)

func TestRootDirNotDist(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	const pluginId = "test-plugin-panel"

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "dist-folder"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:  filepath.Join("testdata", "dist-folder", "dist"),
			metadata.Analyzer: []byte(`{"ID": "` + pluginId + `"}`),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	if assert.Equal(t, 2, interceptor.Counter) {
		assert.Equal(t, interceptor.Diagnostics[0].Title, "Archive root directory named dist. It should contain a directory named test-plugin-panel")
		assert.Equal(t, interceptor.Diagnostics[1].Title, "Archive should contain a directory named test-plugin-panel")
	}
}

func TestRootSameAsPluginId(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	const pluginId = "test-plugin-panel"

	pass := &analysis.Pass{
		RootDir: filepath.Join("testdata", "not-plugin-id"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:  filepath.Join("testdata", "not-plugin-id", "the-files"),
			metadata.Analyzer: []byte(`{"ID": "` + pluginId + `"}`),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	if assert.Equal(t, 1, interceptor.Counter) {
		assert.Equal(t, interceptor.Diagnostics[0].Title, "Archive should contain a directory named test-plugin-panel")
	}

}

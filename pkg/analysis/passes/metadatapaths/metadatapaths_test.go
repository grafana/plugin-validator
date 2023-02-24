package metadatapaths

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/logos"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/screenshots"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestMetadatapathsWithCorrectMetadata(t *testing.T) {
	//prepare logosMeatadata
	var logosMeatadata metadata.MetadataLogos
	require.NoError(t, json.Unmarshal([]byte(`{"small": "img/logo.svg", "large": "img/logo.svg"}`), &logosMeatadata))

	//prepare screenshots
	var screenshotsMetadata []metadata.MetadataScreenshots
	require.NoError(t, json.Unmarshal([]byte(`[{"name": "test", "path": "img/screenshots.png"}]`), &screenshotsMetadata))

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			logos.Analyzer:       logosMeatadata,
			screenshots.Analyzer: screenshotsMetadata,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestMetadatapathsWithWrongLogoPath(t *testing.T) {
	//prepare logosMeatadata
	var logosMeatadata metadata.MetadataLogos
	require.NoError(t, json.Unmarshal([]byte(`{"small": "/img/wrong-with-slash.svg", "large": "./img/wrong-with-dot.svg"}`), &logosMeatadata))

	//prepare screenshots
	var screenshotsMetadata []metadata.MetadataScreenshots
	require.NoError(t, json.Unmarshal([]byte(`[{"name": "test", "path": "img/screenshots.png"}]`), &screenshotsMetadata))
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			logos.Analyzer:       logosMeatadata,
			screenshots.Analyzer: screenshotsMetadata,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	require.Contains(t, interceptor.Diagnostics[0].Title, "plugin.json: relative small logo path should not start with")
	require.Contains(t, interceptor.Diagnostics[1].Title, "plugin.json: relative large logo path should not start with")
}

func TestMetadatapathsWithWrongScreenshotPath(t *testing.T) {
	//prepare logosMeatadata
	var logosMeatadata metadata.MetadataLogos
	err := json.Unmarshal([]byte(`{"small": "img/logo.svg", "large": "img/logo.svg"}`), &logosMeatadata)
	require.NoError(t, err)

	//prepare screenshots
	var screenshotsMetadata []metadata.MetadataScreenshots
	err = json.Unmarshal([]byte(`[{"name": "test", "path": "/img/wrong-with-slash.png"}, {"name": "test2", "path":"./img/wrong-with-dot"}]`), &screenshotsMetadata)
	require.NoError(t, err)
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			logos.Analyzer:       logosMeatadata,
			screenshots.Analyzer: screenshotsMetadata,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	require.Contains(t, interceptor.Diagnostics[0].Title, "plugin.json: relative screenshot path should not start with")
	require.Contains(t, interceptor.Diagnostics[1].Title, "plugin.json: relative screenshot path should not start with")
}

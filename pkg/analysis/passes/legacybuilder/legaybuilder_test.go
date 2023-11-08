package legacybuilder

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/packagejson"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestNoToolKitFound(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	parsedPackageJson := packagejson.PackageJson{
		Name:    "Test Plugin",
		Version: "2.1.2",
		Scripts: map[string]string{
			"test":  "echo 'test'",
			"build": "echo 'build'",
		},
	}

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			packagejson.Analyzer: parsedPackageJson,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

// new plugins get an error
func TestNewPluginToolkitFoundInScripts(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	parsedPackageJson := packagejson.PackageJson{
		Name:    "Test Plugin",
		Version: "2.1.2",
		Scripts: map[string]string{
			"build": "grafana-toolkit plugin:build",
		},
	}

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			packagejson.Analyzer: parsedPackageJson,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "The plugin is using a legacy builder (grafana-toolkit)")
	require.Equal(t, interceptor.Diagnostics[0].Detail, "Script `build` uses grafana-toolkit. Toolkit is deprecated and will not be updated to support new releases of Grafana. Please migrate to create-plugin https://grafana.com/developers/plugin-tools/migration-guides/migrate-from-toolkit.")
	require.Equal(t, interceptor.Diagnostics[0].Severity, analysis.Error)

}

// existing plugins get a warning (for now)
func TestExistingPluginToolkitFoundInScripts(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	parsedPackageJson := packagejson.PackageJson{
		Name:    "Test Plugin",
		Version: "2.1.2",
		Scripts: map[string]string{
			"build": "grafana-toolkit plugin:build",
		},
	}

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			packagejson.Analyzer: parsedPackageJson,
			published.Analyzer: &published.PluginStatus{
				Status: "published",
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "The plugin is using a legacy builder (grafana-toolkit)")
	require.Equal(t, interceptor.Diagnostics[0].Detail, "Script `build` uses grafana-toolkit. Toolkit is deprecated and will not be updated to support new releases of Grafana. Please migrate to create-plugin https://grafana.com/developers/plugin-tools/migration-guides/migrate-from-toolkit.")
	require.Equal(t, interceptor.Diagnostics[0].Severity, analysis.Warning)

}

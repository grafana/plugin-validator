package buildtools

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
)

var (
	nonStandardFrontendBuildTooling = &analysis.Rule{
		Name:     "non-standard-frontend-build-tooling",
		Severity: analysis.Error,
	}
	nonStandardBackendBuildTooling = &analysis.Rule{
		Name:     "non-standard-backend-build-tooling",
		Severity: analysis.Error,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "buildtools",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		nonStandardFrontendBuildTooling,
		nonStandardBackendBuildTooling,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Build Tools",
		Description: "Checks that the plugin uses Grafana's standard create-plugin build tooling.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok || sourceCodeDir == "" {
		return nil, nil
	}

	checkWebpack(pass, sourceCodeDir)
	checkMagefile(pass, sourceCodeDir)

	return nil, nil
}

func checkWebpack(pass *analysis.Pass, sourceCodeDir string) {
	webpackPath := filepath.Join(sourceCodeDir, ".config", "webpack", "webpack.config.ts")
	if _, err := os.Stat(webpackPath); os.IsNotExist(err) {
		pass.ReportResult(pass.AnalyzerName, nonStandardFrontendBuildTooling,
			"non-standard frontend build tooling",
			"The plugin does not appear to use Grafana's standard frontend build tooling. Please use create-plugin to scaffold your plugin: https://grafana.com/developers/plugin-tools/")
		return
	}

	b, err := os.ReadFile(webpackPath)
	if err != nil {
		return
	}

	if !strings.Contains(string(b), "@grafana/create-plugin") {
		pass.ReportResult(pass.AnalyzerName, nonStandardFrontendBuildTooling,
			"non-standard frontend build tooling",
			"The plugin does not appear to use Grafana's standard frontend build tooling. Please use create-plugin to scaffold your plugin: https://grafana.com/developers/plugin-tools/")
	}
}

func checkMagefile(pass *analysis.Pass, sourceCodeDir string) {
	goModPath := filepath.Join(sourceCodeDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return
	}

	magefilePath := filepath.Join(sourceCodeDir, "Magefile.go")
	if _, err := os.Stat(magefilePath); os.IsNotExist(err) {
		pass.ReportResult(pass.AnalyzerName, nonStandardBackendBuildTooling,
			"non-standard backend build tooling",
			"The plugin does not appear to use Grafana's standard backend build tooling. Please use create-plugin to scaffold your plugin: https://grafana.com/developers/plugin-tools/")
		return
	}

	b, err := os.ReadFile(magefilePath)
	if err != nil {
		return
	}

	if !strings.Contains(string(b), "grafana-plugin-sdk-go/build") {
		pass.ReportResult(pass.AnalyzerName, nonStandardBackendBuildTooling,
			"non-standard backend build tooling",
			"The plugin does not appear to use Grafana's standard backend build tooling. Please use create-plugin to scaffold your plugin: https://grafana.com/developers/plugin-tools/")
	}
}

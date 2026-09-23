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

	checkBundlerConfig(pass, sourceCodeDir)
	checkMagefile(pass, sourceCodeDir)

	return nil, nil
}

func checkBundlerConfig(pass *analysis.Pass, sourceCodeDir string) {
	configPaths := []string{
		filepath.Join(sourceCodeDir, ".config", "webpack", "webpack.config.ts"),
		filepath.Join(sourceCodeDir, ".config", "rspack", "rspack.config.ts"),
	}

	for _, configPath := range configPaths {
		b, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}
		if strings.Contains(string(b), "@grafana/create-plugin") {
			return
		}
	}

	pass.ReportResult(pass.AnalyzerName, nonStandardFrontendBuildTooling,
		"non-standard frontend build tooling",
		"The plugin does not appear to use Grafana's standard frontend build tooling. Please use create-plugin to scaffold your plugin: https://grafana.com/developers/plugin-tools/")
}

func checkMagefile(pass *analysis.Pass, sourceCodeDir string) {
	goModPath := filepath.Join(sourceCodeDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return
	}

	matches, _ := filepath.Glob(filepath.Join(sourceCodeDir, "[Mm]agefile.go"))
	if len(matches) == 0 {
		pass.ReportResult(pass.AnalyzerName, nonStandardBackendBuildTooling,
			"non-standard backend build tooling",
			"The plugin does not appear to use Grafana's standard backend build tooling. Please use create-plugin to scaffold your plugin: https://grafana.com/developers/plugin-tools/")
		return
	}

	b, err := os.ReadFile(matches[0])
	if err != nil {
		return
	}

	if !strings.Contains(string(b), "grafana-plugin-sdk-go/build") {
		pass.ReportResult(pass.AnalyzerName, nonStandardBackendBuildTooling,
			"non-standard backend build tooling",
			"The plugin does not appear to use Grafana's standard backend build tooling. Please use create-plugin to scaffold your plugin: https://grafana.com/developers/plugin-tools/")
	}
}

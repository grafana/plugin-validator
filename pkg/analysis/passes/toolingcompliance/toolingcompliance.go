package toolingcompliance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/published"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
)

var (
	missingConfigDir = &analysis.Rule{
		Name:     "missing-config-dir",
		Severity: analysis.Error,
	}
	missingGrafanaTooling = &analysis.Rule{
		Name:     "missing-grafana-tooling",
		Severity: analysis.Error,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "toolingcompliance",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer, published.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingConfigDir, missingGrafanaTooling},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Grafana Tooling Compliance",
		Description: "Ensures the plugin uses Grafana's standard plugin tooling (create-plugin).",
	},
}

// ToolingCheck represents the result of the tooling compliance check
type ToolingCheck struct {
	HasConfigDir      bool
	HasGrafanaTooling bool
}

func run(pass *analysis.Pass) (interface{}, error) {
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)

	// If no source code directory is provided, we can't check tooling compliance
	if !ok || sourceCodeDir == "" {
		return nil, nil
	}

	result := &ToolingCheck{}

	// Check for .config directory
	configDir := filepath.Join(sourceCodeDir, ".config")
	if _, err := os.Stat(configDir); err == nil {
		result.HasConfigDir = true
	}

	// Check for @grafana/create-plugin or @grafana/plugin-configs in devDependencies
	result.HasGrafanaTooling = checkGrafanaToolingInPackageJson(sourceCodeDir)

	// Get published status to adjust severity for existing plugins
	publishedStatus, ok := pass.ResultOf[published.Analyzer].(*published.PluginStatus)
	isPublished := ok && publishedStatus.Status != "unknown"

	// If the plugin is already published, reduce severity to warning
	if isPublished {
		missingConfigDir.Severity = analysis.Warning
		missingGrafanaTooling.Severity = analysis.Warning
	}

	// Report if .config directory is missing
	if !result.HasConfigDir {
		pass.ReportResult(
			pass.AnalyzerName,
			missingConfigDir,
			"Missing .config directory",
			"The plugin source code is missing the .config directory. This indicates the plugin was not created using Grafana's create-plugin tool. Please use https://grafana.com/developers/plugin-tools/ to create and maintain your plugin.",
		)
	}

	// Report if no Grafana tooling is detected
	if !result.HasGrafanaTooling {
		pass.ReportResult(
			pass.AnalyzerName,
			missingGrafanaTooling,
			"Plugin not using Grafana plugin tooling",
			fmt.Sprintf("The plugin's package.json does not include @grafana/create-plugin or @grafana/plugin-configs in devDependencies. Plugins should be built using Grafana's official tooling. Please see https://grafana.com/developers/plugin-tools/get-started/set-up-development-environment for setup instructions."),
		)
	}

	return result, nil
}

// checkGrafanaToolingInPackageJson checks if the package.json contains Grafana tooling in devDependencies
func checkGrafanaToolingInPackageJson(sourceCodeDir string) bool {
	packageJsonPath := filepath.Join(sourceCodeDir, "package.json")
	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return false
	}

	var packageJson struct {
		DevDependencies map[string]string `json:"devDependencies"`
		Dependencies    map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &packageJson); err != nil {
		return false
	}

	// List of Grafana tooling packages that indicate proper tooling usage
	grafanaToolingPackages := []string{
		"@grafana/create-plugin",
		"@grafana/plugin-configs",
		"@grafana/eslint-config",
		"@grafana/tsconfig",
	}

	// Check devDependencies for any of the Grafana tooling packages
	for _, pkg := range grafanaToolingPackages {
		if _, ok := packageJson.DevDependencies[pkg]; ok {
			return true
		}
		// Also check regular dependencies as a fallback
		if _, ok := packageJson.Dependencies[pkg]; ok {
			return true
		}
	}

	return false
}

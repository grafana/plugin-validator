package toolingcompliance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

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
	invalidWebpackConfig = &analysis.Rule{
		Name:     "invalid-webpack-config",
		Severity: analysis.Warning,
	}
	invalidTsConfig = &analysis.Rule{
		Name:     "invalid-tsconfig",
		Severity: analysis.Warning,
	}
	missingStandardScripts = &analysis.Rule{
		Name:     "missing-standard-scripts",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "toolingcompliance",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer, published.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		missingConfigDir,
		missingGrafanaTooling,
		invalidWebpackConfig,
		invalidTsConfig,
		missingStandardScripts,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Grafana Tooling Compliance",
		Description: "Ensures the plugin uses Grafana's standard plugin tooling (create-plugin).",
	},
}

// ToolingCheck represents the result of the tooling compliance check
type ToolingCheck struct {
	HasConfigDir           bool
	HasGrafanaTooling      bool
	HasValidWebpackConfig  bool
	HasValidTsConfig       bool
	HasStandardScripts     bool
	MissingScripts         []string
	ToolingDeviationScore  int // 0 = fully compliant, higher = more deviation
}

// Standard scripts expected in a create-plugin project
var standardScripts = []string{"dev", "build", "test", "lint"}

func run(pass *analysis.Pass) (interface{}, error) {
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)

	// If no source code directory is provided, we can't check tooling compliance
	if !ok || sourceCodeDir == "" {
		return nil, nil
	}

	result := &ToolingCheck{}

	// Get published status to adjust severity for existing plugins
	publishedStatus, ok := pass.ResultOf[published.Analyzer].(*published.PluginStatus)
	isPublished := ok && publishedStatus.Status != "unknown"

	// If the plugin is already published, reduce severity to warning
	if isPublished {
		missingConfigDir.Severity = analysis.Warning
		missingGrafanaTooling.Severity = analysis.Warning
	}

	// Check 1: .config directory presence
	configDir := filepath.Join(sourceCodeDir, ".config")
	if _, err := os.Stat(configDir); err == nil {
		result.HasConfigDir = true
	} else {
		result.ToolingDeviationScore += 3
		pass.ReportResult(
			pass.AnalyzerName,
			missingConfigDir,
			"Missing .config directory",
			"The plugin source code is missing the .config directory. This indicates the plugin was not created using Grafana's create-plugin tool. Please use https://grafana.com/developers/plugin-tools/ to create and maintain your plugin.",
		)
	}

	// Check 2: Grafana tooling packages in package.json
	packageJsonPath := filepath.Join(sourceCodeDir, "package.json")
	packageJson, err := parsePackageJson(packageJsonPath)
	if err == nil {
		result.HasGrafanaTooling = checkGrafanaToolingPackages(packageJson)
		if !result.HasGrafanaTooling {
			result.ToolingDeviationScore += 3
			pass.ReportResult(
				pass.AnalyzerName,
				missingGrafanaTooling,
				"Plugin not using Grafana plugin tooling",
				"The plugin's package.json does not include @grafana/create-plugin or related tooling packages in devDependencies. Plugins should be built using Grafana's official tooling. Please see https://grafana.com/developers/plugin-tools/get-started/set-up-development-environment for setup instructions.",
			)
		}

		// Check 3: Standard scripts in package.json
		result.MissingScripts = checkStandardScripts(packageJson)
		if len(result.MissingScripts) == 0 {
			result.HasStandardScripts = true
		} else {
			result.ToolingDeviationScore++
			pass.ReportResult(
				pass.AnalyzerName,
				missingStandardScripts,
				"Missing standard package.json scripts",
				"The plugin's package.json is missing some standard scripts: "+strings.Join(result.MissingScripts, ", ")+". Plugins created with create-plugin include scripts for dev, build, test, and lint. See https://grafana.com/developers/plugin-tools/ for more information.",
			)
		}
	}

	// Check 4: webpack.config.ts extends from .config
	result.HasValidWebpackConfig = checkWebpackConfig(sourceCodeDir)
	if !result.HasValidWebpackConfig && result.HasConfigDir {
		// Only report if .config exists but webpack doesn't extend from it
		result.ToolingDeviationScore++
		pass.ReportResult(
			pass.AnalyzerName,
			invalidWebpackConfig,
			"webpack.config.ts does not extend from .config",
			"The plugin has a .config directory but webpack.config.ts does not import from './.config/webpack/webpack.config.ts'. This indicates the build configuration may not be using Grafana's standard tooling. See https://grafana.com/developers/plugin-tools/ for the expected configuration.",
		)
	}

	// Check 5: tsconfig.json extends from .config
	result.HasValidTsConfig = checkTsConfig(sourceCodeDir)
	if !result.HasValidTsConfig && result.HasConfigDir {
		// Only report if .config exists but tsconfig doesn't extend from it
		result.ToolingDeviationScore++
		pass.ReportResult(
			pass.AnalyzerName,
			invalidTsConfig,
			"tsconfig.json does not extend from .config",
			"The plugin has a .config directory but tsconfig.json does not extend from './.config/tsconfig.json'. This indicates the TypeScript configuration may not be using Grafana's standard tooling. See https://grafana.com/developers/plugin-tools/ for the expected configuration.",
		)
	}

	return result, nil
}

// PackageJsonFull represents the full package.json structure we need
type PackageJsonFull struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Scripts         map[string]string `json:"scripts"`
	DevDependencies map[string]string `json:"devDependencies"`
	Dependencies    map[string]string `json:"dependencies"`
}

func parsePackageJson(path string) (*PackageJsonFull, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var packageJson PackageJsonFull
	if err := json.Unmarshal(data, &packageJson); err != nil {
		return nil, err
	}

	return &packageJson, nil
}

func checkGrafanaToolingPackages(packageJson *PackageJsonFull) bool {
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

func checkStandardScripts(packageJson *PackageJsonFull) []string {
	var missing []string
	for _, script := range standardScripts {
		if _, ok := packageJson.Scripts[script]; !ok {
			missing = append(missing, script)
		}
	}
	return missing
}

func checkWebpackConfig(sourceCodeDir string) bool {
	// Check for webpack.config.ts or webpack.config.js
	webpackConfigPaths := []string{
		filepath.Join(sourceCodeDir, "webpack.config.ts"),
		filepath.Join(sourceCodeDir, "webpack.config.js"),
	}

	for _, configPath := range webpackConfigPaths {
		content, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}

		contentStr := string(content)
		// Check if it imports from .config/webpack
		if strings.Contains(contentStr, "./.config/webpack") ||
			strings.Contains(contentStr, ".config/webpack") ||
			strings.Contains(contentStr, "@grafana/plugin-configs") {
			return true
		}
	}

	return false
}

func checkTsConfig(sourceCodeDir string) bool {
	tsconfigPath := filepath.Join(sourceCodeDir, "tsconfig.json")
	content, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return false
	}

	contentStr := string(content)
	// Check if it extends from .config/tsconfig.json or uses @grafana/tsconfig
	if strings.Contains(contentStr, "./.config/tsconfig.json") ||
		strings.Contains(contentStr, ".config/tsconfig.json") ||
		strings.Contains(contentStr, "@grafana/tsconfig") {
		return true
	}

	return false
}

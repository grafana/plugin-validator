package levitate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	levitateNotInstalled = &analysis.Rule{Name: "levitate-not-installed", Severity: analysis.Warning}
	apiIncompatibility   = &analysis.Rule{Name: "api-incompatibility", Severity: analysis.Error}
	apiCompatible        = &analysis.Rule{Name: "api-compatible", Severity: analysis.OK}
	invalidGrafanaDep    = &analysis.Rule{Name: "invalid-grafana-dependency", Severity: analysis.Warning}
)

var (
	simpleVersionRe   = regexp.MustCompile(`^(\d+\.\d+\.\d+)`)
	incompatibilityRe = regexp.MustCompile(`^\d+\)\s+(Removed|Changed|Added)\s+` + "`" + `([^` + "`" + `]+)` + "`" + `\s+used in\s+` + "`" + `([^` + "`" + `]+)` + "`")
)

func incrementPatchVersion(version string) (string, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch()+1), nil
}

// versionPatterns extracts minimum version from grafanaDependency constraints.
// Order matters: more specific patterns first.
var versionPatterns = []struct {
	re        *regexp.Regexp
	transform func(string) (string, error)
}{
	{regexp.MustCompile(`>=\s*(\d+\.\d+\.\d+)`), nil},
	{regexp.MustCompile(`>\s*(\d+\.\d+\.\d+)`), incrementPatchVersion},
	{regexp.MustCompile(`^[~^=]\s*(\d+\.\d+\.\d+)`), nil},
	{regexp.MustCompile(`^(\d+)\.[x\*]`), func(v string) (string, error) { return v + ".0.0", nil }},
	{simpleVersionRe, nil},
}

var Analyzer = &analysis.Analyzer{
	Name:     "levitate",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{levitateNotInstalled, apiIncompatibility, apiCompatible, invalidGrafanaDep},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "API Compatibility",
		Description:  "Checks if plugin source code is compatible with target Grafana version ranges using Levitate.",
		Dependencies: "[levitate](https://github.com/grafana/levitate), `sourceCodeUri`",
	},
}

type pluginJSON struct {
	Dependencies struct {
		GrafanaDependency string `json:"grafanaDependency"`
	} `json:"dependencies"`
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func readPluginJSON(sourceCodeDir string) (*pluginJSON, error) {
	paths := []string{
		filepath.Join(sourceCodeDir, "src", "plugin.json"),
		filepath.Join(sourceCodeDir, "plugin.json"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var pj pluginJSON
		if err := json.Unmarshal(data, &pj); err != nil {
			return nil, fmt.Errorf("failed to parse plugin.json: %w", err)
		}
		return &pj, nil
	}
	return nil, fmt.Errorf("plugin.json not found in src/ or root")
}

func readPackageJSON(sourceCodeDir string) (*packageJSON, error) {
	data, err := os.ReadFile(filepath.Join(sourceCodeDir, "package.json"))
	if err != nil {
		return nil, err
	}
	var pj packageJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return nil, err
	}
	return &pj, nil
}

func getGrafanaPackageVersions(pkg *packageJSON) map[string]string {
	versions := make(map[string]string)
	for name, version := range pkg.Dependencies {
		if strings.HasPrefix(name, "@grafana/") {
			versions[name] = version
		}
	}
	for name, version := range pkg.DevDependencies {
		if strings.HasPrefix(name, "@grafana/") && versions[name] == "" {
			versions[name] = version
		}
	}
	return versions
}

func getGrafanaPackageNames(pkg *packageJSON) []string {
	versions := getGrafanaPackageVersions(pkg)
	names := make([]string, 0, len(versions))
	for name := range versions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}


func findModuleEntryPoint(sourceCodeDir string) (string, error) {
	candidates := []string{
		filepath.Join(sourceCodeDir, "src", "module.tsx"),
		filepath.Join(sourceCodeDir, "src", "module.ts"),
		filepath.Join(sourceCodeDir, "module.tsx"),
		filepath.Join(sourceCodeDir, "module.ts"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("module.ts(x) not found")
}

var defaultPackages = []string{
	"@grafana/data",
	"@grafana/ui",
	"@grafana/runtime",
	"@grafana/schema",
	"@grafana/e2e-selectors",
}

func run(pass *analysis.Pass) (any, error) {
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		logme.Debugln("Source code not available, skipping levitate analysis")
		return nil, nil
	}

	minVersion, grafanaDep, err := getMinGrafanaVersion(sourceCodeDir)
	if err != nil {
		logme.DebugFln("Skipping levitate analysis: %v", err)
		return nil, nil
	}
	if minVersion == "" {
		// grafanaDep parse error - report it
		pass.ReportResult(
			pass.AnalyzerName,
			invalidGrafanaDep,
			fmt.Sprintf("Invalid grafanaDependency: %s", grafanaDep),
			fmt.Sprintf("Could not parse version constraint: %v", err),
		)
		return nil, nil
	}

	versionInfo := collectVersionInfo(sourceCodeDir, minVersion)

	// Get @grafana/* packages from package.json, fall back to defaults
	packages := defaultPackages
	if pkg, err := readPackageJSON(sourceCodeDir); err == nil {
		if pkgNames := getGrafanaPackageNames(pkg); len(pkgNames) > 0 {
			packages = pkgNames
		}
	}

	outputStr, err := runLevitate(pass, sourceCodeDir, minVersion, packages)
	if err != nil {
		return nil, nil
	}

	incompatibilities := parseIncompatibilities(outputStr)

	// Report results
	if len(incompatibilities) > 0 {
		for _, incompat := range incompatibilities {
			title, detail := buildIncompatibilityReport(incompat, minVersion, versionInfo)
			pass.ReportResult(pass.AnalyzerName, apiIncompatibility, title, detail)
		}
	} else if apiCompatible.ReportAll {
		pass.ReportResult(
			pass.AnalyzerName,
			apiCompatible,
			fmt.Sprintf("API compatible with Grafana %s", minVersion),
			"No incompatibilities detected",
		)
	}

	return nil, nil
}

func getMinGrafanaVersion(sourceCodeDir string) (string, string, error) {
	pluginJSON, err := readPluginJSON(sourceCodeDir)
	if err != nil {
		return "", "", fmt.Errorf("could not read plugin.json: %w", err)
	}

	grafanaDep := pluginJSON.Dependencies.GrafanaDependency
	if grafanaDep == "" {
		return "", "", fmt.Errorf("no grafanaDependency specified")
	}

	minVersion, err := parseMinVersion(grafanaDep)
	if err != nil {
		return "", grafanaDep, err
	}

	return minVersion, grafanaDep, nil
}

func runLevitate(pass *analysis.Pass, sourceCodeDir, minVersion string, packages []string) (string, error) {
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		if levitateNotInstalled.ReportAll {
			pass.ReportResult(
				pass.AnalyzerName,
				levitateNotInstalled,
				"npx not installed",
				"Skipping API compatibility check. Install Node.js to enable this check.",
			)
		}
		return "", err
	}

	sourcePath, err := findModuleEntryPoint(sourceCodeDir)
	if err != nil {
		logme.DebugFln("Could not find module entry point: %v", err)
		return "", err
	}

	targetArg := strings.Join(buildTargetPackages(packages, minVersion), ",")

	logme.DebugFln("Running levitate compatibility check against Grafana %s", minVersion)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, npxPath, "@grafana/levitate", "is-compatible",
		"--path", sourcePath, "--target", targetArg)
	cmd.Dir = sourceCodeDir

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		logme.ErrorF("Levitate timed out after 2 minutes")
		return "", ctx.Err()
	}

	// levitate returns non-zero exit code if incompatibilities found
	// So we need to parse output even on error
	if err != nil && !strings.Contains(outputStr, "INCOMPATIBILITIES") {
		logme.ErrorF("Error running levitate: %v\nOutput: %s", err, outputStr)
		return "", err
	}

	return outputStr, nil
}

func buildIncompatibilityReport(incompat incompatibility, minVersion string, info *versionInfo) (string, string) {
	userFriendlyType := map[string]string{
		"Removed": "requires newer Grafana version",
		"Changed": "has incompatible changes",
		"Added":   "not available in minimum version",
	}[incompat.changeType]
	if userFriendlyType == "" {
		userFriendlyType = strings.ToLower(incompat.changeType)
	}

	title := fmt.Sprintf("%s: %s %s", incompat.location, incompat.apiName, userFriendlyType)

	var detailParts []string

	if !info.hasVersionMismatch || info.maxPackageVersion == nil {
		detailParts = append(detailParts,
			fmt.Sprintf("Your plugin uses %s which %s in Grafana %s.",
				incompat.apiName, userFriendlyType, minVersion))
	} else {
		pkgVersion := info.maxPackageVersion.String()
		isAddedOrRemoved := incompat.changeType == "Removed" || incompat.changeType == "Added"

		if isAddedOrRemoved {
			detailParts = append(detailParts,
				fmt.Sprintf("Your plugin uses %s which was added in a version newer than Grafana %s.",
					incompat.apiName, minVersion))
		} else {
			detailParts = append(detailParts,
				fmt.Sprintf("Your plugin uses %s which has breaking changes between Grafana %s and %s.",
					incompat.apiName, minVersion, pkgVersion))
		}

		detailParts = append(detailParts,
			fmt.Sprintf("**Version Mismatch:**\n"+
				"- grafanaDependency: >=%s (minimum supported version)\n"+
				"- package.json: @grafana packages at %s",
				minVersion, pkgVersion))

		if isAddedOrRemoved {
			detailParts = append(detailParts,
				fmt.Sprintf("**Recommendation:** Update grafanaDependency in plugin.json to >=%s to match your package.json dependencies, or avoid using %s.",
					pkgVersion, incompat.apiName))
		} else {
			detailParts = append(detailParts,
				fmt.Sprintf("**Recommendation:** Either update grafanaDependency to >=%s or ensure your code is compatible with Grafana %s.",
					pkgVersion, minVersion))
		}
	}

	return title, strings.Join(detailParts, "\n\n")
}

func parseMinVersion(constraint string) (string, error) {
	// Try each pattern to extract a minimum version
	var minVersion string
	for _, p := range versionPatterns {
		matches := p.re.FindStringSubmatch(constraint)
		if len(matches) <= 1 {
			continue
		}
		version := matches[1]
		if p.transform != nil {
			var err error
			minVersion, err = p.transform(version)
			if err != nil {
				return "", err
			}
		} else {
			minVersion = version
		}
		break
	}

	if minVersion == "" {
		return "", fmt.Errorf("unsupported constraint format: %s", constraint)
	}

	// Validate extracted version satisfies the constraint
	parsedVersion, err := semver.NewVersion(minVersion)
	if err != nil {
		return "", err
	}
	semverConstraint, err := semver.NewConstraint(constraint)
	if err != nil {
		return "", err
	}
	if !semverConstraint.Check(parsedVersion) {
		return "", fmt.Errorf("extracted version %s does not satisfy constraint %s", minVersion, constraint)
	}

	return minVersion, nil
}

func buildTargetPackages(packages []string, version string) []string {
	result := make([]string, len(packages))
	for i, pkg := range packages {
		result[i] = fmt.Sprintf("%s@%s", pkg, version)
	}
	return result
}

type incompatibility struct {
	changeType string // "Removed", "Changed", "Added"
	apiName    string
	location   string // file:line
}

type versionInfo struct {
	grafanaDependency     string
	minGrafanaVersion     *semver.Version
	packageVersions       map[string]string
	maxPackageVersion     *semver.Version
	maxPackageVersionName string
	hasVersionMismatch    bool
}

func parseIncompatibilities(output string) []incompatibility {
	var incompatibilities []incompatibility
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := incompatibilityRe.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 4 {
			incompatibilities = append(incompatibilities, incompatibility{
				changeType: matches[1],
				apiName:    matches[2],
				location:   matches[3],
			})
		}
	}

	return incompatibilities
}

func collectVersionInfo(sourceCodeDir string, minGrafanaVersion string) *versionInfo {
	info := &versionInfo{
		grafanaDependency: minGrafanaVersion,
		packageVersions:   make(map[string]string),
	}

	minVersion, err := semver.NewVersion(minGrafanaVersion)
	if err != nil {
		logme.DebugFln("Could not parse minimum Grafana version: %v", err)
		return info
	}
	info.minGrafanaVersion = minVersion

	pkg, err := readPackageJSON(sourceCodeDir)
	if err != nil {
		logme.DebugFln("Could not read package.json: %v", err)
		return info
	}

	grafanaVersions := getGrafanaPackageVersions(pkg)
	info.packageVersions = grafanaVersions

	// Find the maximum package version
	for pkgName, versionConstraint := range grafanaVersions {
		// Extract base version from constraint (e.g., "^10.4.0" -> "10.4.0")
		cleaned := strings.TrimLeft(versionConstraint, "^~>=<")
		if idx := strings.Index(cleaned, " "); idx != -1 {
			cleaned = cleaned[:idx]
		}
		cleaned = strings.TrimSpace(cleaned)

		depVersion, err := semver.NewVersion(cleaned)
		if err != nil {
			continue
		}

		if depVersion.GreaterThan(minVersion) {
			info.hasVersionMismatch = true
		}

		if info.maxPackageVersion == nil || depVersion.GreaterThan(info.maxPackageVersion) {
			info.maxPackageVersion = depVersion
			info.maxPackageVersionName = pkgName
		}
	}

	return info
}

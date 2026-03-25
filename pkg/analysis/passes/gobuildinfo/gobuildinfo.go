package gobuildinfo

import (
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"golang.org/x/mod/modfile"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	binarySourceCodeRequired = &analysis.Rule{
		Name:     "binary-source-code-required",
		Severity: analysis.Warning,
	}
	binaryNoBuildInfo = &analysis.Rule{
		Name:     "binary-no-build-info",
		Severity: analysis.Error,
	}
	binaryDirtyBuild = &analysis.Rule{
		Name:     "binary-dirty-build",
		Severity: analysis.Error,
	}
	binaryPluginIDMismatch = &analysis.Rule{
		Name:     "binary-plugin-id-mismatch",
		Severity: analysis.Error,
	}
	binaryCGOEnabled = &analysis.Rule{
		Name:     "binary-cgo-enabled",
		Severity: analysis.Warning,
	}
	// binary-build-info-json-plugin-id-mismatch: the pluginID field in the SDK's
	// embedded buildInfoJSON does not match the plugin.json ID.
	binaryBuildInfoJSONPluginIDMismatch = &analysis.Rule{
		Name:     "binary-build-info-json-plugin-id-mismatch",
		Severity: analysis.Error,
	}
	// binary-build-info-json-version-mismatch: the version field in the SDK's
	// embedded buildInfoJSON does not match the plugin.json version.
	binaryBuildInfoJSONVersionMismatch = &analysis.Rule{
		Name:     "binary-build-info-json-version-mismatch",
		Severity: analysis.Error,
	}
	// binary-dep-not-in-gomod: a dependency compiled into the binary has no
	// entry in the submitted go.mod.
	binaryDepNotInGoMod = &analysis.Rule{
		Name:     "binary-dep-not-in-gomod",
		Severity: analysis.Error,
	}
	// binary-dep-gomod-version-mismatch: a dependency compiled into the binary
	// is at a different version than declared in the submitted go.mod.
	binaryDepGoModVersionMismatch = &analysis.Rule{
		Name:     "binary-dep-gomod-version-mismatch",
		Severity: analysis.Error,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "gobuildinfo",
	Requires: []*analysis.Analyzer{archive.Analyzer, nestedmetadata.Analyzer, sourcecode.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		binarySourceCodeRequired,
		binaryNoBuildInfo,
		binaryDirtyBuild,
		binaryPluginIDMismatch,
		binaryCGOEnabled,
		binaryBuildInfoJSONPluginIDMismatch,
		binaryBuildInfoJSONVersionMismatch,
		binaryDepNotInGoMod,
		binaryDepGoModVersionMismatch,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Go Build Info",
		Description: "Validates embedded Go build metadata in backend plugin binaries.",
	},
}

// sdkBuildInfo represents the JSON struct embedded via -ldflags by the Grafana
// plugin SDK. Older SDK versions include additional fields (repo, branch, hash,
// build) that are not present in newer versions.
type sdkBuildInfo struct {
	PluginID string `json:"pluginID"`
	Version  string `json:"version"`
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	metadatamap, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		return nil, nil
	}

	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)

	hasBackend := false
	for _, data := range metadatamap {
		if data.Backend && data.Executable != "" {
			hasBackend = true
			break
		}
	}

	if hasBackend && (!ok || sourceCodeDir == "") {
		pass.ReportResult(
			pass.AnalyzerName,
			binarySourceCodeRequired,
			"source code is required to validate backend binaries",
			"Provide the plugin source code to enable backend binary validation checks.",
		)
		return nil, nil
	}

	if !ok || sourceCodeDir == "" {
		return nil, nil
	}

	goMod := parseGoMod(filepath.Join(sourceCodeDir, "go.mod"))

	for pluginJSONPath, data := range metadatamap {
		if !data.Backend || data.Executable == "" {
			continue
		}

		pluginRootDir := filepath.Join(archiveDir, filepath.Dir(pluginJSONPath))
		executableParentDir := filepath.Join(pluginRootDir, filepath.Dir(data.Executable))

		binaries, err := doublestar.FilepathGlob(
			executableParentDir + "/" + filepath.Base(data.Executable) + "*",
		)
		if err != nil {
			logme.Debugln("gobuildinfo: error finding binaries:", err)
			continue
		}

		for _, binary := range binaries {
			checkBinary(pass, binary, data.ID, data.Info.Version, goMod)
		}
	}

	return nil, nil
}

func checkBinary(pass *analysis.Pass, binaryPath, pluginID, pluginVersion string, goMod *modfile.File) {
	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			binaryNoBuildInfo,
			fmt.Sprintf("could not read build info from %s", filepath.Base(binaryPath)),
			"The binary may have been stripped of Go build information. Ensure binaries are built without stripping build metadata.",
		)
		return
	}

	binaryName := filepath.Base(binaryPath)

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.modified":
			if setting.Value == "true" {
				pass.ReportResult(
					pass.AnalyzerName,
					binaryDirtyBuild,
					fmt.Sprintf("%s: built from a dirty working tree", binaryName),
					"The binary was built with uncommitted changes (vcs.modified=true). Binaries submitted for signing should be built from a clean git working tree.",
				)
			}
		case "CGO_ENABLED":
			if setting.Value == "1" {
				pass.ReportResult(
					pass.AnalyzerName,
					binaryCGOEnabled,
					fmt.Sprintf("%s: built with CGO_ENABLED=1", binaryName),
					"Building with CGO enabled makes builds harder to reproduce and verify. Consider building with CGO_ENABLED=0.",
				)
			}
		case "-ldflags":
			checkPluginIDInLDFlags(pass, binaryName, setting.Value, pluginID)
			checkBuildInfoJSON(pass, binaryName, setting.Value, pluginID, pluginVersion)
		}
	}

	if goMod != nil {
		required := make(map[string]string, len(goMod.Require))
		for _, r := range goMod.Require {
			required[r.Mod.Path] = r.Mod.Version
		}
		for _, dep := range info.Deps {
			checkDepGoMod(pass, binaryName, dep, required)
		}
	}
}

func checkPluginIDInLDFlags(pass *analysis.Pass, binaryName, ldflags, expectedID string) {
	_, after, ok := strings.Cut(ldflags, "main.pluginID=")
	if !ok {
		return
	}
	after = strings.TrimPrefix(after, "'")
	if end := strings.IndexAny(after, "' \t"); end >= 0 {
		after = after[:end]
	}
	embeddedID := after
	if embeddedID != expectedID {
		pass.ReportResult(
			pass.AnalyzerName,
			binaryPluginIDMismatch,
			fmt.Sprintf("%s: embedded plugin ID %q does not match plugin.json ID %q", binaryName, embeddedID, expectedID),
			"The plugin ID embedded in the binary at build time does not match the plugin.json ID. Ensure the binary was built for this plugin.",
		)
	}
}

// extractBuildInfoJSON extracts the SDK buildInfoJSON value from ldflags.
// The package path changed between SDK versions:
//   - old: github.com/grafana/grafana-plugin-sdk-go/build.buildInfoJSON
//   - new: github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON
func extractBuildInfoJSON(ldflags string) string {
	for _, prefix := range []string{
		"github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON=",
		"github.com/grafana/grafana-plugin-sdk-go/build.buildInfoJSON=",
	} {
		_, after, ok := strings.Cut(ldflags, prefix)
		if !ok {
			continue
		}
		after = strings.TrimPrefix(after, "'")
		if end := strings.IndexByte(after, '\''); end >= 0 {
			return after[:end]
		}
		return after
	}
	return ""
}

func checkBuildInfoJSON(pass *analysis.Pass, binaryName, ldflags, expectedPluginID, expectedVersion string) {
	raw := extractBuildInfoJSON(ldflags)
	if raw == "" {
		return
	}
	var info sdkBuildInfo
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		return
	}
	if info.PluginID != "" && info.PluginID != expectedPluginID {
		pass.ReportResult(
			pass.AnalyzerName,
			binaryBuildInfoJSONPluginIDMismatch,
			fmt.Sprintf("%s: buildInfoJSON plugin ID %q does not match plugin.json ID %q", binaryName, info.PluginID, expectedPluginID),
			"The plugin ID embedded in the SDK build info JSON does not match the plugin.json ID. Ensure the binary was built for this plugin.",
		)
	}
	if info.Version != "" && expectedVersion != "" && info.Version != expectedVersion {
		pass.ReportResult(
			pass.AnalyzerName,
			binaryBuildInfoJSONVersionMismatch,
			fmt.Sprintf("%s: buildInfoJSON version %q does not match plugin.json version %q", binaryName, info.Version, expectedVersion),
			fmt.Sprintf("The binary was built for version %q but plugin.json declares version %q. Rebuild the backend binary.", info.Version, expectedVersion),
		)
	}
}

// parseGoMod reads and parses a go.mod file.
func parseGoMod(path string) *modfile.File {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	f, err := modfile.ParseLax(path, data, nil)
	if err != nil {
		return nil
	}
	return f
}

func checkDepGoMod(pass *analysis.Pass, binaryName string, dep *debug.Module, goMod map[string]string) {
	if dep.Replace != nil {
		dep = dep.Replace
	}
	modVersion, ok := goMod[dep.Path]
	if !ok {
		pass.ReportResult(
			pass.AnalyzerName,
			binaryDepNotInGoMod,
			fmt.Sprintf("%s: dependency %s@%s is not in go.mod", binaryName, dep.Path, dep.Version),
			fmt.Sprintf(
				"The binary was compiled with %s but it is not declared in go.mod. "+
					"Rebuild the backend binary after updating go.mod.",
				dep.Path,
			),
		)
		return
	}
	if dep.Version != modVersion {
		pass.ReportResult(
			pass.AnalyzerName,
			binaryDepGoModVersionMismatch,
			fmt.Sprintf("%s: dependency %s version mismatch: binary has %s, go.mod requires %s", binaryName, dep.Path, dep.Version, modVersion),
			fmt.Sprintf(
				"The binary was compiled with %s@%s but go.mod requires %s. "+
					"Rebuild the backend binary.",
				dep.Path, dep.Version, modVersion,
			),
		)
	}
}

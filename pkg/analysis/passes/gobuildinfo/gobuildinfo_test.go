package gobuildinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

// buildTestBinary compiles a minimal Go binary into dir and returns its path.
// The binary embeds pluginID via -ldflags so checkPluginIDInLDFlags can be exercised.
func buildTestBinary(t *testing.T, dir, pluginID string) string {
	t.Helper()
	src := filepath.Join(dir, "main.go")
	err := os.WriteFile(src, []byte(`package main
import _ "runtime/debug"
func main() {}
`), 0644)
	require.NoError(t, err)

	out := filepath.Join(dir, "gpx_test_linux_amd64")
	ldflags := "-X 'main.pluginID=" + pluginID + "'"
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", out, src)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("build output: %s", output)
	}
	require.NoError(t, err)
	return out
}

func makePass(t *testing.T, archiveDir, sourceDir string, meta nestedmetadata.Metadatamap) (*analysis.Pass, *testpassinterceptor.TestPassInterceptor) {
	t.Helper()
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:      archiveDir,
			nestedmetadata.Analyzer: meta,
			sourcecode.Analyzer:   sourceDir,
		},
		Report: interceptor.ReportInterceptor(),
	}
	return pass, &interceptor
}

func TestNonBackendPluginSkipped(t *testing.T) {
	meta := nestedmetadata.Metadatamap{
		"plugin.json": {ID: "test-plugin", Backend: false},
	}
	pass, interceptor := makePass(t, t.TempDir(), "", meta)
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Empty(t, interceptor.Diagnostics)
}

func TestCleanBinary(t *testing.T) {
	dir := t.TempDir()
	pluginID := "myorg-myplugin-datasource"
	binary := buildTestBinary(t, dir, pluginID)

	meta := nestedmetadata.Metadatamap{
		"plugin.json": {ID: pluginID, Backend: true, Executable: "gpx_test"},
	}
	pass, interceptor := makePass(t, dir, "", meta)
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	// A clean local build may have vcs.modified=true or no vcs info at all — only
	// assert there's no plugin-id-mismatch or go-sum-mismatch.
	for _, d := range interceptor.Diagnostics {
		require.NotEqual(t, "binary-plugin-id-mismatch", d.Name)
		require.NotEqual(t, "binary-go-sum-mismatch", d.Name)
	}
	_ = binary
}


func TestPluginIDMismatch(t *testing.T) {
	dir := t.TempDir()
	binary := buildTestBinary(t, dir, "myorg-otherplugin-datasource")

	meta := nestedmetadata.Metadatamap{
		"plugin.json": {ID: "myorg-myplugin-datasource", Backend: true, Executable: "gpx_test"},
	}
	pass, interceptor := makePass(t, dir, dir, meta)
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)

	var names []string
	for _, d := range interceptor.Diagnostics {
		names = append(names, d.Name)
	}
	require.Contains(t, names, "binary-plugin-id-mismatch")
	_ = binary
}

func TestNoBuildInfo(t *testing.T) {
	dir := t.TempDir()
	// Write a file that is not a Go binary.
	notABinary := filepath.Join(dir, "gpx_fake_linux_amd64")
	err := os.WriteFile(notABinary, []byte("not a go binary"), 0755)
	require.NoError(t, err)

	meta := nestedmetadata.Metadatamap{
		"plugin.json": {ID: "myorg-fake-datasource", Backend: true, Executable: "gpx_fake"},
	}
	pass, interceptor := makePass(t, dir, dir, meta)
	_, err = Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "binary-no-build-info", interceptor.Diagnostics[0].Name)
}


func TestCheckPluginIDInLDFlagsMatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	ldflags := `-w -s -X 'main.pluginID=myorg-myplugin-datasource' -X 'main.version=1.0.0'`
	checkPluginIDInLDFlags(pass, "gpx_test", ldflags, "myorg-myplugin-datasource")
	require.Empty(t, interceptor.Diagnostics)
}

func TestCheckPluginIDInLDFlagsMismatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	ldflags := `-w -s -X 'main.pluginID=myorg-otherplugin-datasource'`
	checkPluginIDInLDFlags(pass, "gpx_test", ldflags, "myorg-myplugin-datasource")
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "binary-plugin-id-mismatch", interceptor.Diagnostics[0].Name)
}

func TestCheckBuildInfoJSONMatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	// new SDK path
	ldflags := `-w -s -X 'github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON={"pluginID":"myorg-myplugin-datasource","version":"1.0.0"}'`
	checkBuildInfoJSON(pass, "gpx_test", ldflags, "myorg-myplugin-datasource", "1.0.0")
	require.Empty(t, interceptor.Diagnostics)
}

func TestCheckBuildInfoJSONOldSDKMatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	// old SDK path with extra fields
	ldflags := `-w -s -X 'github.com/grafana/grafana-plugin-sdk-go/build.buildInfoJSON={"time":1714395852089,"pluginID":"myorg-myplugin-datasource","version":"1.0.0","repo":"https://github.com/example/repo","branch":"main","hash":"abc123","build":42}'`
	checkBuildInfoJSON(pass, "gpx_test", ldflags, "myorg-myplugin-datasource", "1.0.0")
	require.Empty(t, interceptor.Diagnostics)
}

func TestCheckBuildInfoJSONPluginIDMismatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	ldflags := `-w -s -X 'github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON={"pluginID":"myorg-otherplugin-datasource","version":"1.0.0"}'`
	checkBuildInfoJSON(pass, "gpx_test", ldflags, "myorg-myplugin-datasource", "1.0.0")
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "binary-build-info-json-plugin-id-mismatch", interceptor.Diagnostics[0].Name)
}

func TestCheckBuildInfoJSONVersionMismatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	ldflags := `-w -s -X 'github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON={"pluginID":"myorg-myplugin-datasource","version":"1.0.0"}'`
	checkBuildInfoJSON(pass, "gpx_test", ldflags, "myorg-myplugin-datasource", "2.0.0")
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "binary-build-info-json-version-mismatch", interceptor.Diagnostics[0].Name)
}

func TestCheckBuildInfoJSONAbsent(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	// binary built without SDK (no buildInfoJSON) — should be a no-op
	ldflags := `-w -s -X 'main.pluginID=myorg-myplugin-datasource'`
	checkBuildInfoJSON(pass, "gpx_test", ldflags, "myorg-myplugin-datasource", "1.0.0")
	require.Empty(t, interceptor.Diagnostics)
}

func TestParseGoMod(t *testing.T) {
	dir := t.TempDir()
	content := `module github.com/example/plugin

go 1.21

require (
	github.com/foo/bar v1.2.3
	github.com/baz/qux v0.1.0 // indirect
)
`
	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644)
	require.NoError(t, err)

	f := parseGoMod(filepath.Join(dir, "go.mod"))
	require.NotNil(t, f)
	versions := make(map[string]string)
	for _, r := range f.Require {
		versions[r.Mod.Path] = r.Mod.Version
	}
	require.Equal(t, "v1.2.3", versions["github.com/foo/bar"])
	require.Equal(t, "v0.1.0", versions["github.com/baz/qux"])
}

func TestCheckDepGoModMatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	dep := &debug.Module{Path: "github.com/foo/bar", Version: "v1.2.3"}
	goMod := map[string]string{"github.com/foo/bar": "v1.2.3"}
	checkDepGoMod(pass, "gpx_test", dep, goMod)
	require.Empty(t, interceptor.Diagnostics)
}

func TestCheckDepGoModNotInGoMod(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	dep := &debug.Module{Path: "github.com/foo/bar", Version: "v1.2.3"}
	goMod := map[string]string{}
	checkDepGoMod(pass, "gpx_test", dep, goMod)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "binary-dep-not-in-gomod", interceptor.Diagnostics[0].Name)
}

func TestCheckDepGoModVersionMismatch(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	dep := &debug.Module{Path: "github.com/foo/bar", Version: "v1.3.0"}
	goMod := map[string]string{"github.com/foo/bar": "v1.2.3"}
	checkDepGoMod(pass, "gpx_test", dep, goMod)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "binary-dep-gomod-version-mismatch", interceptor.Diagnostics[0].Name)
}

func TestCheckDepGoModReplace(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Report: interceptor.ReportInterceptor()}
	pass.AnalyzerName = Analyzer.Name

	// Replace directive: check is against the replacement module
	dep := &debug.Module{
		Path:    "github.com/foo/bar",
		Version: "v1.2.3",
		Replace: &debug.Module{Path: "github.com/fork/bar", Version: "v1.2.3"},
	}
	goMod := map[string]string{"github.com/fork/bar": "v1.2.3"}
	checkDepGoMod(pass, "gpx_test", dep, goMod)
	require.Empty(t, interceptor.Diagnostics)
}


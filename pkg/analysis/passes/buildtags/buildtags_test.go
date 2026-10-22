package buildtags

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/runner"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/grafana/plugin-validator/pkg/testutils"
)

const pluginJSON = `{
	"id": "test-plugin-datasource",
	"type": "datasource",
	"backend": true,
	"executable": "gpx_plugin"
}`

// fixtureDir holds fixture binaries shared across tests so each distinct
// build happens only once per test run.
var fixtureDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "buildtags-fixtures")
	if err != nil {
		panic(err)
	}
	fixtureDir = dir
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

type buildOpts struct {
	goos, goarch string
	tags         string // value passed to -tags, if any
	goflags      string // value of the GOFLAGS environment variable
}

// fixtureBinary builds (once) a minimal Go program with the given options and
// returns its path.
func fixtureBinary(t *testing.T, name string, opts buildOpts) string {
	t.Helper()
	destPath := filepath.Join(fixtureDir, name)
	if _, err := os.Stat(destPath); err == nil {
		return destPath
	}

	modDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module testbin\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	// Strip and trim like release builds, to show the tag survives in the build info.
	args := []string{"build", "-trimpath", "-ldflags=-s -w", "-o", destPath}
	if opts.tags != "" {
		args = append(args, "-tags", opts.tags)
	}
	args = append(args, ".")
	cmd := exec.Command("go", args...)
	cmd.Dir = modDir
	// Later entries override inherited ones, so GOFLAGS from the environment is never used.
	cmd.Env = append(os.Environ(),
		"GOOS="+opts.goos,
		"GOARCH="+opts.goarch,
		"CGO_ENABLED=0",
		"GOFLAGS="+opts.goflags,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build test binary: %s", out)
	return destPath
}

func linuxBinary(t *testing.T, name, tags, goflags string) string {
	return fixtureBinary(t, name, buildOpts{goos: "linux", goarch: "amd64", tags: tags, goflags: goflags})
}

func copyFile(t *testing.T, src, dest string) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(dest), 0o755))
	require.NoError(t, os.WriteFile(dest, data, 0o755))
}

func runAnalyzer(t *testing.T, archiveDir string, metadata map[string]string) []*analysis.Diagnostic {
	t.Helper()
	metadatamap := nestedmetadata.Metadatamap{}
	for path, content := range metadata {
		meta, err := testutils.JSONToMetadata([]byte(content))
		require.NoError(t, err)
		metadatamap[path] = meta
	}

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		Analyzer:     Analyzer,
		AnalyzerName: Analyzer.Name,
		RootDir:      archiveDir,
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:        archiveDir,
			nestedmetadata.Analyzer: metadatamap,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	return interceptor.Diagnostics
}

func TestReadBuildTags(t *testing.T) {
	for _, tt := range []struct {
		name string
		tags string
		want []string
	}{
		{name: "no tags"},
		{name: "comma separated", tags: "custom_tag,another_tag", want: []string{"custom_tag", "another_tag"}},
		{name: "space separated", tags: "custom_tag another_tag", want: []string{"custom_tag", "another_tag"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tags, err := readBuildTags(linuxBinary(t, "tags-"+tt.name, tt.tags, ""))
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, tags)
		})
	}

	t.Run("unreadable build info", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "not-go")
		require.NoError(t, os.WriteFile(path, []byte("not a Go binary"), 0o644))
		_, err := readBuildTags(path)
		require.Error(t, err)
	})
}

func TestBuildTagRuleConfiguration(t *testing.T) {
	originalRule := *marketplaceDevBuild
	t.Cleanup(func() { *marketplaceDevBuild = originalRule })

	archiveDir := t.TempDir()
	pluginDir := filepath.Join(archiveDir, "test-plugin-datasource")
	copyFile(t, linuxBinary(t, "dev", "marketplace_dev", ""), filepath.Join(pluginDir, "gpx_plugin_linux_amd64"))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0o644))

	for _, tt := range []struct {
		name     string
		config   string
		severity analysis.Severity
	}{
		{name: "disabled by default"},
		{
			name:     "enabled by rule",
			config:   "analyzers:\n  buildtags:\n    rules:\n      marketplace-dev-build:\n        enabled: true\n",
			severity: analysis.Error,
		},
		{
			name:     "enabled by analyzer",
			config:   "analyzers:\n  buildtags:\n    enabled: true\n",
			severity: analysis.Error,
		},
		{
			name:   "rule overrides analyzer",
			config: "analyzers:\n  buildtags:\n    enabled: true\n    rules:\n      marketplace-dev-build:\n        enabled: false\n",
		},
		{
			name:     "severity override",
			config:   "analyzers:\n  buildtags:\n    rules:\n      marketplace-dev-build:\n        enabled: true\n        severity: warning\n",
			severity: analysis.Warning,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var config runner.Config
			require.NoError(t, yaml.Unmarshal([]byte("global:\n  enabled: true\n"+tt.config), &config))
			diagnostics, err := runner.Check(
				[]*analysis.Analyzer{Analyzer},
				analysis.CheckParams{ArchiveDir: archiveDir},
				config,
				"",
			)
			require.NoError(t, err)
			if tt.severity == "" {
				require.Empty(t, diagnostics[Analyzer.Name])
				return
			}
			require.Len(t, diagnostics[Analyzer.Name], 1)
			diagnostic := diagnostics[Analyzer.Name][0]
			require.Equal(t, "marketplace-dev-build", diagnostic.Name)
			require.Equal(t, tt.severity, diagnostic.Severity)
			require.Contains(t, diagnostic.Detail, "gpx_plugin_linux_amd64")
		})
	}
}

func TestSupportsAdditionalBuildTagRules(t *testing.T) {
	originalChecks := buildTagChecks
	t.Cleanup(func() { buildTagChecks = originalChecks })
	buildTagChecks = append(buildTagChecks, buildTagCheck{
		tag:      "custom_tag",
		rule:     &analysis.Rule{Name: "custom-tag-build", Severity: analysis.Warning},
		title:    "Backend executable built with custom tag",
		guidance: "Rebuild without custom_tag.",
	})

	archiveDir := t.TempDir()
	copyFile(t, linuxBinary(t, "dev", "marketplace_dev", ""), filepath.Join(archiveDir, "gpx_plugin_dev"))
	copyFile(t, linuxBinary(t, "custom", "custom_tag", ""), filepath.Join(archiveDir, "gpx_plugin_custom"))

	diagnostics := runAnalyzer(t, archiveDir, map[string]string{"plugin.json": pluginJSON})
	require.ElementsMatch(t, []*analysis.Diagnostic{
		{
			Name:     "marketplace-dev-build",
			Severity: analysis.Error,
			Title:    "Backend executable built with marketplace development mode",
			Detail:   fmt.Sprintf("The backend executable gpx_plugin_dev was built with the %q build tag. %s", "marketplace_dev", buildTagChecks[0].guidance),
		},
		{
			Name:     "custom-tag-build",
			Severity: analysis.Warning,
			Title:    "Backend executable built with custom tag",
			Detail:   "The backend executable gpx_plugin_custom was built with the \"custom_tag\" build tag. Rebuild without custom_tag.",
		},
	}, diagnostics)
}

func TestDetectsMarketplaceDevTag(t *testing.T) {
	tests := []struct {
		name   string
		binary func(t *testing.T) string
	}{
		{
			name:   "tags flag",
			binary: func(t *testing.T) string { return linuxBinary(t, "dev", "marketplace_dev", "") },
		},
		{
			name:   "GOFLAGS",
			binary: func(t *testing.T) string { return linuxBinary(t, "dev-goflags", "", "-tags=marketplace_dev") },
		},
		{
			name: "multiple tags",
			binary: func(t *testing.T) string {
				return linuxBinary(t, "dev-multi", "arrow_json_stdlib,marketplace_dev", "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archiveDir := t.TempDir()
			copyFile(t, tt.binary(t), filepath.Join(archiveDir, "gpx_plugin_linux_amd64"))

			diagnostics := runAnalyzer(t, archiveDir, map[string]string{"plugin.json": pluginJSON})
			require.Len(t, diagnostics, 1)
			require.Equal(t, "marketplace-dev-build", diagnostics[0].Name)
			require.Equal(t, analysis.Error, diagnostics[0].Severity)
			require.Equal(t, "Backend executable built with marketplace development mode", diagnostics[0].Title)
			require.Contains(t, diagnostics[0].Detail, "gpx_plugin_linux_amd64")
		})
	}
}

func TestIgnoresBinariesWithoutMarketplaceDevTag(t *testing.T) {
	tests := []struct {
		name   string
		binary func(t *testing.T, dest string)
	}{
		{
			name: "no tags",
			binary: func(t *testing.T, dest string) {
				copyFile(t, linuxBinary(t, "plain", "", ""), dest)
			},
		},
		{
			name: "similar tag name",
			binary: func(t *testing.T, dest string) {
				copyFile(t, linuxBinary(t, "dev-tools", "marketplace_dev_tools", ""), dest)
			},
		},
		{
			name: "non-Go file",
			binary: func(t *testing.T, dest string) {
				require.NoError(t, os.WriteFile(dest, []byte("not a binary"), 0o755))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archiveDir := t.TempDir()
			tt.binary(t, filepath.Join(archiveDir, "gpx_plugin_linux_amd64"))

			diagnostics := runAnalyzer(t, archiveDir, map[string]string{"plugin.json": pluginJSON})
			require.Empty(t, diagnostics)
		})
	}
}

func TestIgnoresPluginWithoutExecutable(t *testing.T) {
	archiveDir := t.TempDir()
	copyFile(t, linuxBinary(t, "dev", "marketplace_dev", ""), filepath.Join(archiveDir, "gpx_plugin_linux_amd64"))

	diagnostics := runAnalyzer(t, archiveDir, map[string]string{
		"plugin.json": `{"id": "test-plugin-panel", "type": "panel"}`,
	})
	require.Empty(t, diagnostics)
}

func TestDetectsNestedPlugin(t *testing.T) {
	archiveDir := t.TempDir()
	copyFile(t, linuxBinary(t, "dev", "marketplace_dev", ""), filepath.Join(archiveDir, "nested", "gpx_plugin_linux_amd64"))

	diagnostics := runAnalyzer(t, archiveDir, map[string]string{
		"plugin.json":        `{"id": "test-plugin-app", "type": "app"}`,
		"nested/plugin.json": pluginJSON,
	})
	require.Len(t, diagnostics, 1)
	require.Contains(t, diagnostics[0].Detail, "nested/gpx_plugin_linux_amd64")
}

func TestReportsOncePerBinary(t *testing.T) {
	archiveDir := t.TempDir()
	copyFile(t, linuxBinary(t, "dev", "marketplace_dev", ""), filepath.Join(archiveDir, "gpx_plugin_linux_amd64"))
	darwin := fixtureBinary(t, "dev-darwin", buildOpts{goos: "darwin", goarch: "arm64", tags: "marketplace_dev"})
	copyFile(t, darwin, filepath.Join(archiveDir, "gpx_plugin_darwin_arm64"))

	diagnostics := runAnalyzer(t, archiveDir, map[string]string{"plugin.json": pluginJSON})
	check := buildTagChecks[0]
	var expected []*analysis.Diagnostic
	for _, binary := range []string{"gpx_plugin_darwin_arm64", "gpx_plugin_linux_amd64"} {
		expected = append(expected, &analysis.Diagnostic{
			Name:     "marketplace-dev-build",
			Title:    "Backend executable built with marketplace development mode",
			Severity: analysis.Error,
			Detail:   fmt.Sprintf("The backend executable %s was built with the %q build tag. %s", binary, check.tag, check.guidance),
		})
	}
	require.ElementsMatch(t, expected, diagnostics)
}

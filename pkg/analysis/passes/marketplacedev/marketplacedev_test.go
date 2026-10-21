package marketplacedev

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
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
	dir, err := os.MkdirTemp("", "marketplacedev-fixtures")
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
		RootDir: archiveDir,
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

func TestReportsOnceForMultipleBinaries(t *testing.T) {
	archiveDir := t.TempDir()
	copyFile(t, linuxBinary(t, "dev", "marketplace_dev", ""), filepath.Join(archiveDir, "gpx_plugin_linux_amd64"))
	darwin := fixtureBinary(t, "dev-darwin", buildOpts{goos: "darwin", goarch: "arm64", tags: "marketplace_dev"})
	copyFile(t, darwin, filepath.Join(archiveDir, "gpx_plugin_darwin_arm64"))

	diagnostics := runAnalyzer(t, archiveDir, map[string]string{"plugin.json": pluginJSON})
	require.Len(t, diagnostics, 1)
	require.Contains(t, diagnostics[0].Detail, "gpx_plugin_darwin_arm64, gpx_plugin_linux_amd64")
}

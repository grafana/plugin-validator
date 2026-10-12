package marketplacelicense

import (
	"debug/buildinfo"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/runner"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestMarketplaceBuildSignals(t *testing.T) {
	tests := []struct {
		name          string
		goos          string
		goarch        string
		tags          string
		includeMarker bool
		wantRule      string
		wantDetail    string
	}{
		{
			name:   "readable metadata without tags",
			goos:   "linux",
			goarch: "amd64",
		},
		{
			name:       "tag only",
			goos:       "darwin",
			goarch:     "arm64",
			tags:       marketplaceDevelopmentTag,
			wantRule:   marketplaceDevelopmentBuild.Name,
			wantDetail: "build tag",
		},
		{
			name:          "marker only",
			goos:          "windows",
			goarch:        "amd64",
			includeMarker: true,
			wantRule:      marketplaceDevelopmentBuild.Name,
			wantDetail:    "development marker",
		},
		{
			name:          "tag and marker",
			goos:          "linux",
			goarch:        "arm64",
			tags:          "other," + marketplaceDevelopmentTag,
			includeMarker: true,
			wantRule:      marketplaceDevelopmentBuild.Name,
			wantDetail:    " and ",
		},
		{
			name:   "tag prefix is not an exact match",
			goos:   "linux",
			goarch: "amd64",
			tags:   marketplaceDevelopmentTag + "_extra,other",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archiveDir := t.TempDir()
			binaryPath := filepath.Join(archiveDir, executableName(test.goos, test.goarch))
			buildTestBinary(t, binaryPath, test.goos, test.goarch, test.tags, test.includeMarker)

			diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
				nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
			})
			if test.wantRule == "" {
				require.Empty(t, diagnostics)
				return
			}
			require.Len(t, diagnostics, 1)
			require.Equal(t, test.wantRule, diagnostics[0].Name)
			require.Contains(t, diagnostics[0].Detail, test.wantDetail)
		})
	}
}

func TestUnreadableBuildMetadata(t *testing.T) {
	t.Run("without marker requires manual review", func(t *testing.T) {
		archiveDir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(archiveDir, "marketplace_test_linux_amd64"),
			[]byte("not a Go executable"),
			0o755,
		))

		diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		})
		require.Len(t, diagnostics, 1)
		require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
	})

	t.Run("marker still rejects", func(t *testing.T) {
		archiveDir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(archiveDir, "marketplace_test_linux_amd64"),
			[]byte("prefix"+marketplaceDevelopmentMarker+"suffix"),
			0o755,
		))

		diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		})
		require.Len(t, diagnostics, 1)
		require.Equal(t, marketplaceDevelopmentBuild.Name, diagnostics[0].Name)
		require.Contains(t, diagnostics[0].Detail, "development marker")
	})
}

func TestMixedPlatformArchiveInspectsEveryBinary(t *testing.T) {
	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, executableName("linux", "amd64")),
		"linux",
		"amd64",
		"",
		false,
	)
	buildTestBinary(
		t,
		filepath.Join(archiveDir, executableName("windows", "amd64")),
		"windows",
		"amd64",
		marketplaceDevelopmentTag,
		true,
	)

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
	})
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceDevelopmentBuild.Name, diagnostics[0].Name)
	require.Contains(t, diagnostics[0].Title, "marketplace_test_windows_amd64.exe")
}

func TestNestedPluginExecutablesAreInspected(t *testing.T) {
	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, executableName("linux", "amd64")),
		"linux",
		"amd64",
		"",
		false,
	)
	buildTestBinary(
		t,
		filepath.Join(archiveDir, "nested", executableName("darwin", "arm64")),
		"darwin",
		"arm64",
		marketplaceDevelopmentTag,
		true,
	)

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		"nested/plugin.json":          backendMetadata("marketplace_test"),
	})
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceDevelopmentBuild.Name, diagnostics[0].Name)
	require.Contains(t, diagnostics[0].Title, "nested/marketplace_test_darwin_arm64")
}

func TestRunnerDiscoversPackagedBackendExecutables(t *testing.T) {
	t.Cleanup(func() {
		for _, rule := range Analyzer.Rules {
			rule.Disabled = false
		}
	})

	extractedArchiveDir := t.TempDir()
	pluginDir := filepath.Join(extractedArchiveDir, "myorg-marketplace-datasource")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pluginDir, "plugin.json"),
		[]byte(`{"id":"myorg-marketplace-datasource","type":"datasource","backend":true,"executable":"marketplace_test"}`),
		0o644,
	))
	buildTestBinary(
		t,
		filepath.Join(pluginDir, executableName("linux", "amd64")),
		"linux",
		"amd64",
		marketplaceDevelopmentTag,
		true,
	)

	sourceDir := t.TempDir()
	writeMarketplaceSourceFixture(t, sourceDir)

	diagnostics, err := runner.Check(
		[]*analysis.Analyzer{Analyzer},
		analysis.CheckParams{
			ArchiveDir:    extractedArchiveDir,
			SourceCodeDir: sourceDir,
		},
		runner.Config{Global: runner.GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)
	require.NoError(t, err)
	require.Len(t, diagnostics[Analyzer.Name], 1)
	require.Equal(t, marketplaceDevelopmentBuild.Name, diagnostics[Analyzer.Name][0].Name)
	require.Contains(t, diagnostics[Analyzer.Name][0].Title, "marketplace_test_linux_amd64")

	disabled := false
	diagnostics, err = runner.Check(
		[]*analysis.Analyzer{Analyzer},
		analysis.CheckParams{
			ArchiveDir:    extractedArchiveDir,
			SourceCodeDir: sourceDir,
		},
		runner.Config{
			Global: runner.GlobalConfig{Enabled: true},
			Analyzers: map[string]runner.AnalyzerConfig{
				Analyzer.Name: {Enabled: &disabled},
			},
		},
		analysis.Severity(""),
	)
	require.NoError(t, err)
	require.Empty(t, diagnostics[Analyzer.Name])
}

func TestOrdinaryBackendDoesNotRequireMarketplaceEntrypoint(t *testing.T) {
	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, executableName("linux", "amd64")),
		"linux",
		"amd64",
		"",
		false,
	)
	sourceDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "main.go"),
		[]byte("package main\nfunc main() {}\n"),
		0o644,
	))

	diagnostics := runAnalyzerWithSource(
		t,
		archiveDir,
		map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		},
		sourceDir,
	)
	require.Empty(t, diagnostics)
}

func TestDevelopmentBuildIsRejectedWithoutMarketplaceClassification(t *testing.T) {
	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, executableName("linux", "amd64")),
		"linux",
		"amd64",
		marketplaceDevelopmentTag,
		true,
	)

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
	})
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceDevelopmentBuild.Name, diagnostics[0].Name)
}

func TestMissingAnalysisDependenciesFailClosed(t *testing.T) {
	tests := []struct {
		name     string
		resultOf map[*analysis.Analyzer]interface{}
	}{
		{
			name: "archive result",
			resultOf: map[*analysis.Analyzer]interface{}{
				nestedmetadata.Analyzer: nestedmetadata.Metadatamap{},
			},
		},
		{
			name: "metadata result",
			resultOf: map[*analysis.Analyzer]interface{}{
				archive.Analyzer: t.TempDir(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{
				AnalyzerName: Analyzer.Name,
				ResultOf:     test.resultOf,
				Report:       interceptor.ReportInterceptor(),
			}

			_, err := Analyzer.Run(pass)
			require.NoError(t, err)
			require.Len(t, interceptor.Diagnostics, 1)
			require.Equal(t, marketplaceBuildUnverified.Name, interceptor.Diagnostics[0].Name)
		})
	}
}

func TestBackendExecutableDiscoveryFailsClosed(t *testing.T) {
	t.Run("missing executable declaration", func(t *testing.T) {
		diagnostics := runAnalyzer(t, t.TempDir(), map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: {Backend: true},
		})
		require.Len(t, diagnostics, 1)
		require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
		require.Contains(t, diagnostics[0].Detail, "does not declare an executable")
	})

	t.Run("missing packaged executable", func(t *testing.T) {
		diagnostics := runAnalyzer(t, t.TempDir(), map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		})
		require.Len(t, diagnostics, 1)
		require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
		require.Contains(t, diagnostics[0].Detail, "No packaged backend executable matched")
	})

	t.Run("executable with backend disabled", func(t *testing.T) {
		diagnostics := runAnalyzer(t, t.TempDir(), map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: {
				Backend:    false,
				Executable: "marketplace_test",
			},
		})
		require.Len(t, diagnostics, 1)
		require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
		require.Contains(t, diagnostics[0].Detail, "sets backend to false")
	})
}

func TestBuildManifestIsNotTreatedAsAnExecutable(t *testing.T) {
	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, "go_plugin_linux_amd64"),
		"linux",
		"amd64",
		"",
		false,
	)
	require.NoError(t, os.WriteFile(
		filepath.Join(archiveDir, "go_plugin_build_manifest"),
		[]byte("manifest"),
		0o644,
	))

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("go_plugin"),
	})
	require.Empty(t, diagnostics)
}

func TestExecutableNameTreatsGlobMetacharactersLiterally(t *testing.T) {
	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, "marketplace[abc]_linux_amd64"),
		"linux",
		"amd64",
		"",
		false,
	)
	require.NoError(t, os.WriteFile(
		filepath.Join(archiveDir, "marketplacea_linux_amd64"),
		[]byte(marketplaceDevelopmentMarker),
		0o755,
	))

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("marketplace[abc]"),
	})
	require.Empty(t, diagnostics)
}

func TestExecutablePathCannotEscapeArchive(t *testing.T) {
	parentDir := t.TempDir()
	archiveDir := filepath.Join(parentDir, "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(parentDir, "escaped_linux_amd64"),
		[]byte(marketplaceDevelopmentMarker),
		0o755,
	))

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("../escaped"),
	})
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
	require.NotContains(t, diagnostics[0].Title, "development build")
}

func TestExecutableSymlinkCannotEscapeArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires additional privileges on Windows")
	}

	parentDir := t.TempDir()
	archiveDir := filepath.Join(parentDir, "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	outsidePath := filepath.Join(parentDir, "outside")
	require.NoError(t, os.WriteFile(outsidePath, []byte(marketplaceDevelopmentMarker), 0o755))
	require.NoError(t, os.Symlink(
		outsidePath,
		filepath.Join(archiveDir, "marketplace_test_linux_amd64"),
	))

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
	})
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
	require.Contains(t, diagnostics[0].Detail, "symlink resolves outside")
}

func TestExecutableDirectorySymlinkCannotEscapeArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires additional privileges on Windows")
	}

	parentDir := t.TempDir()
	archiveDir := filepath.Join(parentDir, "archive")
	outsideDir := filepath.Join(parentDir, "outside")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, os.MkdirAll(outsideDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(outsideDir, "marketplace_test_linux_amd64"),
		[]byte(marketplaceDevelopmentMarker),
		0o755,
	))
	require.NoError(t, os.Symlink(outsideDir, filepath.Join(archiveDir, "bin")))

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("bin/marketplace_test"),
	})
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
	require.Contains(t, diagnostics[0].Detail, "directory symlink resolves outside")
}

func TestNonRegularExecutableIsNotScanned(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix domain socket fixture is not portable to Windows")
	}

	archiveDir := t.TempDir()
	socketPath := filepath.Join(archiveDir, "x")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, listener.Close()) })

	diagnostics := runAnalyzer(t, archiveDir, map[string]metadata.Metadata{
		nestedmetadata.MainPluginJson: backendMetadata("x"),
	})
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
	require.Contains(t, diagnostics[0].Detail, "not a regular file")
}

func TestMarketplaceSourceSymlinkCannotEscapeCheckout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires additional privileges on Windows")
	}

	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, executableName("linux", "amd64")),
		"linux",
		"amd64",
		"",
		false,
	)
	sourceDir := t.TempDir()
	writeMarketplaceSourceFixture(t, sourceDir)
	require.NoError(t, os.Remove(filepath.Join(sourceDir, "main.go")))
	require.NoError(t, os.Symlink("/dev/zero", filepath.Join(sourceDir, "main.go")))

	diagnostics := runAnalyzerWithSource(
		t,
		archiveDir,
		map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		},
		sourceDir,
	)
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
	require.Contains(t, diagnostics[0].Detail, "symlink resolves outside")
}

func TestSDKDependencyUsesEffectiveReplacementVersion(t *testing.T) {
	found, version, replaced := inspectSDKDependency(&buildinfo.BuildInfo{
		Deps: []*debug.Module{
			{
				Path:    marketplaceSDKModule,
				Version: "v1.0.0",
				Replace: &debug.Module{
					Path:    marketplaceSDKModule,
					Version: "v2.0.0",
				},
			},
		},
	})
	require.True(t, found)
	require.Equal(t, "v2.0.0", version)
	require.False(t, replaced)
}

func TestSourceSDKUsesVersionSpecificReplacementBeforeWildcard(t *testing.T) {
	archiveDir := t.TempDir()
	binaryPath := filepath.Join(archiveDir, executableName("linux", "amd64"))
	buildTestBinary(
		t,
		binaryPath,
		"linux",
		"amd64",
		"",
		false,
	)
	binaryBuildInfo, err := buildinfo.ReadFile(binaryPath)
	require.NoError(t, err)
	found, binarySDKVersion, replaced := inspectSDKDependency(binaryBuildInfo)
	require.True(t, found)
	require.False(t, replaced)
	require.NotEmpty(t, binarySDKVersion)

	sourceDir := t.TempDir()
	writeMarketplaceSourceFixture(t, sourceDir)
	goMod := fmt.Sprintf(`module marketplace-validator-test

go 1.21

require %[1]s v1.2.3
replace %[1]s => %[1]s v0.0.1
replace %[1]s v1.2.3 => %[1]s %[2]s
`, marketplaceSDKModule, binarySDKVersion)
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "go.mod"), []byte(goMod), 0o644))

	diagnostics := runAnalyzerWithSource(
		t,
		archiveDir,
		map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		},
		sourceDir,
	)
	require.Empty(t, diagnostics)
}

func TestVendoredMarketplaceSDKRequiresIntegrityReview(t *testing.T) {
	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, executableName("linux", "amd64")),
		"linux",
		"amd64",
		"",
		false,
	)
	sourceDir := t.TempDir()
	writeMarketplaceSourceFixture(t, sourceDir)
	require.NoError(t, os.MkdirAll(
		filepath.Join(sourceDir, "vendor", filepath.FromSlash(marketplaceSDKModule)),
		0o755,
	))

	diagnostics := runAnalyzerWithSource(
		t,
		archiveDir,
		map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		},
		sourceDir,
	)
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
	require.Contains(t, diagnostics[0].Detail, "integrity review")
}

func TestMarketplaceSourceAndBinarySDKMustMatch(t *testing.T) {
	tests := []struct {
		name        string
		goModSuffix string
		wantDetail  string
	}{
		{
			name:        "source SDK replacement",
			goModSuffix: "\nreplace github.com/grafana/grafana-plugin-sdk-go => ../fork\n",
			wantDetail:  "local path or fork",
		},
		{
			name:       "source and binary SDK versions differ",
			wantDetail: "source requires Grafana Plugin SDK v1.2.3",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archiveDir := t.TempDir()
			buildTestBinary(
				t,
				filepath.Join(archiveDir, executableName("linux", "amd64")),
				"linux",
				"amd64",
				"",
				false,
			)
			sourceDir := t.TempDir()
			writeMarketplaceSourceFixture(t, sourceDir)
			if test.goModSuffix != "" {
				contents, err := os.ReadFile(filepath.Join(sourceDir, "go.mod"))
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(
					filepath.Join(sourceDir, "go.mod"),
					append(contents, test.goModSuffix...),
					0o644,
				))
			} else {
				require.NoError(t, os.WriteFile(
					filepath.Join(sourceDir, "go.mod"),
					[]byte("module marketplace-validator-test\n\ngo 1.21\n\nrequire github.com/grafana/grafana-plugin-sdk-go v1.2.3\n"),
					0o644,
				))
			}

			diagnostics := runAnalyzerWithSource(
				t,
				archiveDir,
				map[string]metadata.Metadata{
					nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
				},
				sourceDir,
			)
			require.NotEmpty(t, diagnostics)
			require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
			require.Contains(t, diagnostics[0].Detail, test.wantDetail)
		})
	}
}

func TestMarketplaceSourceEntrypointDetection(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{
			name: "Manage with alias",
			source: `package main
import licensing "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func main() { licensing.Manage("plugin-id") }
`,
			want: true,
		},
		{
			name: "direct checker with dot import",
			source: `package main
import . "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func main() { CheckMarketplacePluginLicense("plugin-id") }
`,
			want: true,
		},
		{
			name: "import without enforcement call",
			source: `package main
import _ "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func main() {}
`,
		},
		{
			name: "unreachable call",
			source: `package main
import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func unused() { marketplace.Manage("plugin-id") }
func main() {}
`,
		},
		{
			name: "development-only entrypoint",
			source: `//go:build marketplace_dev

package main
import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func main() { marketplace.Manage("plugin-id") }
`,
		},
		{
			name: "reachable wrapper",
			source: `package main
import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func enforce() { marketplace.Manage("plugin-id") }
func main() { enforce() }
`,
			want: true,
		},
		{
			name: "shadowed wrapper",
			source: `package main
import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func enforce() { marketplace.Manage("plugin-id") }
func main() {
	enforce := func() {}
	enforce()
}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sourceDir := t.TempDir()
			require.NoError(t, os.WriteFile(
				filepath.Join(sourceDir, "main.go"),
				[]byte(test.source),
				0o644,
			))
			_, found, err := sourceUsesMarketplaceLicense(sourceDir, targetBuild{
				goos:   "linux",
				goarch: "amd64",
				tags:   make(map[string]bool),
			})
			require.NoError(t, err)
			require.Equal(t, test.want, found)
		})
	}
}

func TestSourceUsesMarketplaceLicenseMatchesTargetBuild(t *testing.T) {
	sourceDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "main.go"),
		[]byte("package main\nfunc main() {}\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "license_windows.go"),
		[]byte(`package main
import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func init() { marketplace.Manage("plugin-id") }
`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "license_ignore.go"),
		[]byte(`//go:build ignore

package main
import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func init() { marketplace.Manage("plugin-id") }
`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "license_cgo.go"),
		[]byte(`package main
import "C"
import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func init() { marketplace.Manage("plugin-id") }
`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "license_amd64.go"),
		[]byte(`//go:build !amd64.v1

package main
import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
func init() { marketplace.Manage("plugin-id") }
`),
		0o644,
	))

	_, linux, err := sourceUsesMarketplaceLicense(sourceDir, targetBuild{
		goos:   "linux",
		goarch: "amd64",
		tags:   map[string]bool{"amd64.v1": true},
	})
	require.NoError(t, err)
	require.False(t, linux)

	_, windows, err := sourceUsesMarketplaceLicense(sourceDir, targetBuild{
		goos:   "windows",
		goarch: "amd64",
		tags:   map[string]bool{"amd64.v1": true},
	})
	require.NoError(t, err)
	require.True(t, windows)
}

func TestMarketplaceSourceFIFOIsRejectedBeforeOpening(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipes require a Unix-like system")
	}

	archiveDir := t.TempDir()
	buildTestBinary(
		t,
		filepath.Join(archiveDir, executableName("linux", "amd64")),
		"linux",
		"amd64",
		"",
		false,
	)
	sourceDir := t.TempDir()
	writeMarketplaceSourceFixture(t, sourceDir)
	require.NoError(t, os.Remove(filepath.Join(sourceDir, "main.go")))
	require.NoError(t, exec.Command("mkfifo", filepath.Join(sourceDir, "main.go")).Run())

	diagnostics := runAnalyzerWithSource(
		t,
		archiveDir,
		map[string]metadata.Metadata{
			nestedmetadata.MainPluginJson: backendMetadata("marketplace_test"),
		},
		sourceDir,
	)
	require.Len(t, diagnostics, 1)
	require.Equal(t, marketplaceBuildUnverified.Name, diagnostics[0].Name)
	require.Contains(t, diagnostics[0].Detail, "not a regular file")
}

func TestMarkerScanAcrossBufferBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "binary")
	prefix := strings.Repeat("x", markerScanBufferSize-len(marketplaceDevelopmentMarker)/2)
	require.NoError(t, os.WriteFile(
		path,
		[]byte(prefix+marketplaceDevelopmentMarker+"suffix"),
		0o644,
	))

	found, err := fileContainsMarker(path, []byte(marketplaceDevelopmentMarker))
	require.NoError(t, err)
	require.True(t, found)
}

func runAnalyzer(
	t *testing.T,
	archiveDir string,
	metadataMap nestedmetadata.Metadatamap,
) []*analysis.Diagnostic {
	t.Helper()
	sourceDir := t.TempDir()
	writeMarketplaceSourceFixture(t, sourceDir)

	return runAnalyzerWithSource(t, archiveDir, metadataMap, sourceDir)
}

func runAnalyzerWithSource(
	t *testing.T,
	archiveDir string,
	metadataMap nestedmetadata.Metadatamap,
	sourceDir string,
) []*analysis.Diagnostic {
	t.Helper()
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:        archiveDir,
			nestedmetadata.Analyzer: metadataMap,
			sourcecode.Analyzer:     sourceDir,
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	return interceptor.Diagnostics
}

func writeMarketplaceSourceFixture(t *testing.T, sourceDir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "go.mod"),
		[]byte("module marketplace-validator-test\n\ngo 1.21\n\nrequire github.com/grafana/grafana-plugin-sdk-go v0.0.0\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "main.go"),
		[]byte(`package main

import "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"

func main() { marketplace.Manage("plugin-id") }
`),
		0o644,
	))
}

func backendMetadata(executable string) metadata.Metadata {
	return metadata.Metadata{
		Backend:    true,
		Executable: executable,
	}
}

func executableName(goos, goarch string) string {
	name := "marketplace_test_" + goos + "_" + goarch
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func buildTestBinary(
	t *testing.T,
	destination string,
	goos string,
	goarch string,
	tags string,
	includeMarker bool,
) {
	t.Helper()
	moduleDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(moduleDir, "go.mod"),
		[]byte("module marketplace-validator-test\n\ngo 1.21\n\nrequire github.com/grafana/grafana-plugin-sdk-go v0.0.0\n"),
		0o644,
	))
	vendorSDKDir := filepath.Join(
		moduleDir,
		"vendor",
		"github.com",
		"grafana",
		"grafana-plugin-sdk-go",
	)
	require.NoError(t, os.MkdirAll(vendorSDKDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(moduleDir, "vendor", "modules.txt"),
		[]byte("# github.com/grafana/grafana-plugin-sdk-go v0.0.0\n## explicit; go 1.21\ngithub.com/grafana/grafana-plugin-sdk-go\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(vendorSDKDir, "sdk.go"),
		[]byte("package sdk\n\nfunc Keep() {}\n"),
		0o644,
	))

	source := "package main\n\nimport sdk \"github.com/grafana/grafana-plugin-sdk-go\"\n\nfunc main() { sdk.Keep() }\n"
	if includeMarker {
		source = "package main\n\nimport (\n\t\"fmt\"\n\tsdk \"github.com/grafana/grafana-plugin-sdk-go\"\n)\n\nconst marker = \"" +
			marketplaceDevelopmentMarker + "\"\n\nfunc main() { sdk.Keep(); fmt.Print(marker) }\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte(source), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))

	args := []string{"build", "-mod=vendor", "-trimpath", "-ldflags=-s -w", "-o", destination}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	args = append(args, ".")
	cmd := exec.Command("go", args...)
	cmd.Dir = moduleDir
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build test binary: %s", output)
}

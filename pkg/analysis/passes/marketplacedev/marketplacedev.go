package marketplacedev

import (
	"debug/buildinfo"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/logme"
)

// marketplaceDevTag is the plugin SDK build tag that disables marketplace license checks.
const marketplaceDevTag = "marketplace_dev"

var marketplaceDevBuild = &analysis.Rule{
	Name:     "marketplace-dev-build",
	Severity: analysis.Error,
}

// Analyzer rejects backend executables built with the SDK marketplace_dev build tag.
var Analyzer = &analysis.Analyzer{
	Name:     "marketplacedev",
	Requires: []*analysis.Analyzer{archive.Analyzer, nestedmetadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{marketplaceDevBuild},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Marketplace Dev Mode",
		Description: "Rejects backend executables built with the SDK `marketplace_dev` build tag, which disables marketplace license checks.",
	},
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

	checked := map[string]struct{}{}
	var affected []string
	for pluginJsonPath, data := range metadatamap {
		if data.Executable == "" {
			continue
		}

		executable := data.Executable
		executableParentDir := filepath.Join(
			archiveDir,
			filepath.Dir(pluginJsonPath),
			filepath.Dir(executable),
		)

		foundBinaries, err := doublestar.FilepathGlob(
			executableParentDir + "/" + filepath.Base(executable) + "*",
		)
		if err != nil {
			logme.Debugln("Error walking", executableParentDir, err)
		}

		for _, binaryPath := range foundBinaries {
			if _, ok := checked[binaryPath]; ok {
				continue
			}
			checked[binaryPath] = struct{}{}

			if !hasMarketplaceDevTag(binaryPath) {
				continue
			}
			relPath, err := filepath.Rel(archiveDir, binaryPath)
			if err != nil {
				relPath = binaryPath
			}
			affected = append(affected, filepath.ToSlash(relPath))
		}
	}

	if len(affected) == 0 {
		return nil, nil
	}

	slices.Sort(affected)
	pass.ReportResult(
		pass.AnalyzerName,
		marketplaceDevBuild,
		"Backend executable built with marketplace development mode",
		fmt.Sprintf(
			"The following backend executables were built with the %q build tag, which disables marketplace license checks: %s. Please rebuild them without the %q build tag (for example, use a regular build target instead of build:debugMarketplace).",
			marketplaceDevTag,
			strings.Join(affected, ", "),
			marketplaceDevTag,
		),
	)

	return nil, nil
}

// hasMarketplaceDevTag reports whether the Go build info of the given binary
// records the marketplace_dev build tag. Binaries without readable build info
// (non-Go, or packed/obfuscated) are skipped.
func hasMarketplaceDevTag(binaryPath string) bool {
	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		logme.Debugln("Could not read build info for", binaryPath, err)
		return false
	}
	for _, setting := range info.Settings {
		if setting.Key != "-tags" {
			continue
		}
		tags := strings.FieldsFunc(setting.Value, func(r rune) bool {
			return r == ',' || unicode.IsSpace(r)
		})
		return slices.Contains(tags, marketplaceDevTag)
	}
	return false
}

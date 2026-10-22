package buildtags

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

var marketplaceDevBuild = &analysis.Rule{
	Name:              "marketplace-dev-build",
	Severity:          analysis.Error,
	DisabledByDefault: true,
}

type buildTagCheck struct {
	tag      string
	rule     *analysis.Rule
	title    string
	guidance string
}

// Add checks here and register their rules in Analyzer.Rules.
var buildTagChecks = []buildTagCheck{
	{
		tag:      "marketplace_dev",
		rule:     marketplaceDevBuild,
		title:    "Backend executable built with marketplace development mode",
		guidance: "This tag disables marketplace license checks. Please rebuild without it (for example, use a regular build target instead of build:debugMarketplace).",
	},
}

// Analyzer checks backend executables for disallowed Go build tags.
var Analyzer = &analysis.Analyzer{
	Name:     "buildtags",
	Requires: []*analysis.Analyzer{archive.Analyzer, nestedmetadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{marketplaceDevBuild},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Build Tags",
		Description: "Checks Go backend executables for disallowed build tags. The `marketplace-dev-build` rule rejects `marketplace_dev` and is disabled by default.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Avoid opening binaries when no build-tag rules are enabled.
	hasEnabledRule := slices.ContainsFunc(buildTagChecks, func(check buildTagCheck) bool {
		return !check.rule.Disabled
	})
	if !hasEnabledRule {
		return nil, nil
	}

	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	metadatamap, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		return nil, nil
	}

	checked := map[string]struct{}{}
	affected := map[*analysis.Rule][]string{}
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

			tags, err := readBuildTags(binaryPath)
			if err != nil {
				logme.Debugln("Could not read build info for", binaryPath, err)
				continue
			}
			relPath, err := filepath.Rel(archiveDir, binaryPath)
			if err != nil {
				relPath = binaryPath
			}
			for _, check := range buildTagChecks {
				if !check.rule.Disabled && slices.Contains(tags, check.tag) {
					affected[check.rule] = append(affected[check.rule], filepath.ToSlash(relPath))
				}
			}
		}
	}

	for _, check := range buildTagChecks {
		binaries := affected[check.rule]
		if len(binaries) == 0 {
			continue
		}
		slices.Sort(binaries)
		pass.ReportResult(
			pass.AnalyzerName,
			check.rule,
			check.title,
			fmt.Sprintf(
				"The following backend executables were built with the %q build tag: %s. %s",
				check.tag,
				strings.Join(binaries, ", "),
				check.guidance,
			),
		)
	}

	return nil, nil
}

// readBuildTags returns the explicit Go build tags recorded in a binary's build
// info. It returns an error for binaries without readable Go build info.
func readBuildTags(binaryPath string) ([]string, error) {
	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		return nil, err
	}
	for _, setting := range info.Settings {
		if setting.Key != "-tags" {
			continue
		}
		return strings.FieldsFunc(setting.Value, func(r rune) bool {
			return r == ',' || unicode.IsSpace(r)
		}), nil
	}
	return nil, nil
}

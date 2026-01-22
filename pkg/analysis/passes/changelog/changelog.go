package changelog

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	missingChangelog = &analysis.Rule{Name: "missing-changelog", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "changelog",
	Run:      run,
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Rules:    []*analysis.Rule{missingChangelog},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Changelog (exists)",
		Description: "Ensures a `CHANGELOG.md` file exists within the zip file.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := analysis.GetResult[string](pass, archive.Analyzer)
	if !ok {
		return nil, nil
	}

	b, err := os.ReadFile(filepath.Join(archiveDir, "CHANGELOG.md"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.ReportResult(pass.AnalyzerName, missingChangelog, "missing CHANGELOG.md", "A CHANGELOG.md is missing from the plugin archive.")
			return nil, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		pass.ReportResult(pass.AnalyzerName, missingChangelog, "CHANGELOG.md is empty", "A CHANGELOG.md file is empty.")
		return nil, nil
	}
	if missingChangelog.ReportAll {
		missingChangelog.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, missingChangelog, "CHANGELOG.md: exists and not empty", "")
	}

	return b, nil
}

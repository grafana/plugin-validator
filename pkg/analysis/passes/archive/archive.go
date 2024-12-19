package archive

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
)

var (
	emptyArchive   = &analysis.Rule{Name: "empty-archive", Severity: analysis.Error}
	moreThanOneDir = &analysis.Rule{Name: "more-than-one-dir", Severity: analysis.Error}
	noRootDir      = &analysis.Rule{Name: "no-root-dir", Severity: analysis.Error}
	dist           = &analysis.Rule{Name: "dist", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:  "archive",
	Run:   run,
	Rules: []*analysis.Rule{emptyArchive, moreThanOneDir, noRootDir, dist},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Archive Structure",
		Description: "Ensures the contents of the zip file have the expected layout.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	fis, err := os.ReadDir(pass.RootDir)
	if err != nil {
		return nil, err
	}

	if len(fis) == 0 {
		pass.ReportResult(pass.AnalyzerName, emptyArchive, "Archive is empty", "")
		return nil, nil
	}

	if emptyArchive.ReportAll {
		emptyArchive.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, emptyArchive, "Archive is not empty", "")
	}

	if len(fis) != 1 {
		pass.ReportResult(
			pass.AnalyzerName,
			moreThanOneDir,
			"Archive contains more than one directory",
			fmt.Sprintf(
				"Archive should contain only one directory named after plugin id. Found %d directories. Please see https://grafana.com/developers/plugin-tools/publish-a-plugin/package-a-plugin for more information on how to package a plugin.",
				len(fis),
			),
		)
		return nil, nil
	}
	if moreThanOneDir.ReportAll {
		moreThanOneDir.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, moreThanOneDir, "Archive has a single directory", "")
	}

	if !fis[0].IsDir() {
		pass.ReportResult(
			pass.AnalyzerName,
			noRootDir,
			"archive does not contain a root directory",
			"Archive should contain a single root directory. Found a file instead. Please see https://grafana.com/developers/plugin-tools/publish-a-plugin/package-a-plugin for more information on how to package a plugin.",
		)
		return nil, nil
	}
	if noRootDir.ReportAll {
		noRootDir.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, noRootDir, "Archive contains a root directory", "")
	}

	rootDir := filepath.Join(pass.RootDir, fis[0].Name())
	legacyRoot := filepath.Join(rootDir, "dist")

	_, err = os.Stat(legacyRoot)
	if err != nil {
		if os.IsNotExist(err) {
			if dist.ReportAll {
				dist.Severity = analysis.OK
				pass.ReportResult(pass.AnalyzerName, dist, "Archive has expected content", "")
			}
			return rootDir, nil
		}
		return nil, err
	}

	pass.ReportResult(
		pass.AnalyzerName,
		dist,
		"dist should be renamed to plugin id and moved to root. Please see https://grafana.com/developers/plugin-tools/publish-a-plugin/package-a-plugin for more information on how to package a plugin.",
		"",
	)

	return legacyRoot, nil
}

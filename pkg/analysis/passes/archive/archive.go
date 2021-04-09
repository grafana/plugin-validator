package archive

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
)

var (
	emptyArchive   = &analysis.Rule{Name: "empty-archive"}
	moreThanOneDir = &analysis.Rule{Name: "more-than-one-dir"}
	noRootDir      = &analysis.Rule{Name: "no-root-dir"}
	dist           = &analysis.Rule{Name: "dist"}
)

var Analyzer = &analysis.Analyzer{
	Name:  "archive",
	Run:   run,
	Rules: []*analysis.Rule{emptyArchive, moreThanOneDir, noRootDir, dist},
}

func run(pass *analysis.Pass) (interface{}, error) {
	fis, err := ioutil.ReadDir(pass.RootDir)
	if err != nil {
		return nil, err
	}

	if len(fis) == 0 {
		pass.Reportf(emptyArchive, "archive is empty")
		return nil, nil
	}

	if len(fis) != 1 {
		pass.Reportf(moreThanOneDir, "archive contains more than one directory")
		return nil, nil
	}

	if !fis[0].IsDir() {
		pass.Reportf(noRootDir, "archive does not contain a root directory")
		return nil, nil
	}

	rootDir := filepath.Join(pass.RootDir, fis[0].Name())
	legacyRoot := filepath.Join(rootDir, "dist")

	_, err = os.Stat(legacyRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return rootDir, nil
		}
		return nil, err
	}

	pass.Reportf(dist, "dist should be renamed to plugin id and moved to root")

	return legacyRoot, nil
}

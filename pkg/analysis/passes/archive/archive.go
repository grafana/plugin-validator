package archive

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "archive",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	fis, err := ioutil.ReadDir(pass.RootDir)
	if err != nil {
		return nil, err
	}

	if len(fis) == 0 {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Error,
			Message:  "archive is empty",
		})
		return nil, nil
	}

	if len(fis) != 1 {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Error,
			Message:  "archive contains more than one directory",
		})
		return nil, nil
	}

	if !fis[0].IsDir() {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Error,
			Message:  "archive does not contain a identifying directory",
		})
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

	pass.Report(analysis.Diagnostic{
		Severity: analysis.Error,
		Message:  "dist should be renamed to plugin id and moved to root",
	})

	return legacyRoot, nil
}

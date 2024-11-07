package backenddebug

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
)

var (
	backendDebugFilePresent = &analysis.Rule{
		Name:     "backend-debug-file-present",
		Severity: analysis.Error,
	}
	archiveFilesError = &analysis.Rule{
		Name:     "archive-read-error",
		Severity: analysis.Error,
	}
)

// Analyzer is an analyzer that checks if backend standalone debug files are included in the executable.
// If so, it reports an error, as the plugin can't be used properly in non-debug mode.
var Analyzer = &analysis.Analyzer{
	Name:     "backenddebug",
	Requires: []*analysis.Analyzer{archive.Analyzer, nestedmetadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{backendDebugFilePresent},
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

	hasExecutable := false
	for _, data := range metadatamap {
		if data.Executable != "" {
			hasExecutable = true
			break
		}
	}

	// don't evaluate for plugins that don't have an executable
	if !hasExecutable {
		return nil, nil
	}

	textFiles, err := doublestar.FilepathGlob(archiveDir + "/**/*.txt")
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			archiveFilesError,
			"error reading archive files",
			fmt.Sprintf("error reading archive files: %s", err.Error()),
		)
	}

	for _, file := range textFiles {
		fileName := filepath.Base(file)
		if fileName == "standalone.txt" || fileName == "pid.txt" {
			pass.ReportResult(
				pass.AnalyzerName,
				backendDebugFilePresent,
				"found standalone backend file",
				fmt.Sprintf(
					"You have bundled %q, which will make the plugin unusable in production mode. Please remove it",
					fileName,
				),
			)
		}
	}

	return nil, nil
}

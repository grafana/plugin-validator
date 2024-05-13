package backendbinary

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	backendBinaryMissing = &analysis.Rule{
		Name:     "backend-binary-mission",
		Severity: analysis.Error,
	}
	backendFoundButNotDeclared = &analysis.Rule{
		Name:     "backend-found-but-not-declared",
		Severity: analysis.Error,
	}
	alertingFoundButBackendFalse = &analysis.Rule{
		Name:     "alerting-found-but-backend-false",
		Severity: analysis.Error,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "backendbinary",
	Requires: []*analysis.Analyzer{archive.Analyzer, nestedmetadata.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		backendBinaryMissing,
		backendFoundButNotDeclared,
		alertingFoundButBackendFalse,
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

	for pluginJsonPath, data := range metadatamap {

		// alerting declared without backend true
		if data.Alerting && !data.Backend {
			pass.ReportResult(
				pass.AnalyzerName,
				alertingFoundButBackendFalse,
				"Found alerting in plugin.json but backend=false",
				fmt.Sprintf(
					"You have marked your plugin with backend=false in %s but declared an alerting=true Please set backend=true in your plugin.json if your plugin has a backend component",
					pluginJsonPath,
				),
			)
			return nil, nil
		}

		// executable declared without backend true
		if data.Executable != "" && !data.Backend {
			pass.ReportResult(
				pass.AnalyzerName,
				backendFoundButNotDeclared,
				"Found executable in plugin.json but backend=false",
				fmt.Sprintf(
					"You have marked your plugin with backend=false in %s but declared a backend executable. Please set backend=true in your plugin.json if your plugin has a backend component or remove the executable from your plugin.json",
					pluginJsonPath,
				),
			)
			return nil, nil
		}

		// backend true without executable declared
		if data.Backend && data.Executable == "" {
			pass.ReportResult(
				pass.AnalyzerName,
				backendBinaryMissing,
				"Missing executable in plugin.json",
				fmt.Sprintf(
					"You have marked backend=true in %s but have not added a backend executable. Please add a backend executable to your plugin.json if your plugin has a backend component or set backend=false",
					pluginJsonPath,
				),
			)
			return nil, nil
		}

		// no executable in plugin.json skipping other checks
		if data.Executable == "" {
			return nil, nil
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

		// backend true but no backend binaries found
		if data.Backend && len(foundBinaries) == 0 {
			pass.ReportResult(
				pass.AnalyzerName,
				backendBinaryMissing,
				"Missing backend binaries in your plugin archive",
				fmt.Sprintf(
					"You have declared a backend component in %s.json but have not found any backend binaries. Please add backend binaries to your plugin archive",
					pluginJsonPath,
				),
			)
			return nil, nil
		}
	}

	return nil, nil
}

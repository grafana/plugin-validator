package backendbinary

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var binarySuffixes = []string{
	"_linux_amd64",
	"_linux_arm64",
	"_darwin_amd64",
	"_darwin_arm64",
	"_windows_amd64.exe",
}
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
	Requires: []*analysis.Analyzer{archive.Analyzer, metadata.Analyzer},
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
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	// alerting declared without backend true
	if data.Alerting && !data.Backend {
		pass.ReportResult(
			pass.AnalyzerName,
			alertingFoundButBackendFalse,
			"Found alerting in plugin.json but backend=false",
			"You have marked your plugin with backend=false in your plugin.json but declared an alerting=true Please set backend=true in your plugin.json if your plugin has an alerting component",
		)
		return nil, nil
	}

	// executable declared without backend true
	if data.Executable != "" && !data.Backend {
		pass.ReportResult(
			pass.AnalyzerName,
			backendFoundButNotDeclared,
			"Found executable in plugin.json but backend=false",
			"You have marked your plugin with backend=false in your plugin.json but declared a backend executable. Please set backend=true in your plugin.json if your plugin has a bakend component or remove the executable from your plugin.json",
		)
		return nil, nil
	}

	// backend true without executable declared
	if data.Backend && data.Executable == "" {
		pass.ReportResult(
			pass.AnalyzerName,
			backendBinaryMissing,
			"Missing executable in plugin.json",
			"You have marked backend=true in your plugin.json but have not added a backend executable. Please add a backend executable to your plugin.json if your plugin has a bakend component or set backend=false",
		)
		return nil, nil
	}

	// no executable in plugin.json skipping other checks
	if data.Executable == "" {
		return nil, nil
	}

	executable := data.Executable
	executableParentDir := filepath.Join(archiveDir, filepath.Dir(executable))
	executableName := filepath.Base(executable)

	var foundBinaries = []string{}
	for _, suffix := range binarySuffixes {
		binaryPath := filepath.Join(executableParentDir, executableName+suffix)
		if _, err := os.Stat(binaryPath); err != nil {
			continue
		}
		foundBinaries = append(foundBinaries, binaryPath)
	}

	// backend true but no backend binaries found
	if data.Backend && len(foundBinaries) == 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			backendBinaryMissing,
			"Missing backend binaries in your plugin archive",
			"You have declared a backend component in your plugin.json but have not found any backend binaries. Please add backend binaries to your plugin archive",
		)
		return nil, nil
	}

	return nil, nil
}

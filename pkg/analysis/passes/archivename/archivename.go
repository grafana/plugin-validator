package archivename

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	noIdentRootDir = &analysis.Rule{Name: "no-ident-root-dir", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "archivename",
	Requires: []*analysis.Analyzer{archive.Analyzer, metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{noIdentRootDir},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	base := filepath.Base(archiveDir)

	if base == "dist" {
		pass.ReportResult(pass.AnalyzerName, noIdentRootDir, fmt.Sprintf("Archive root directory named dist. It should contain a directory named %s", data.ID), "The plugin archive file should contain a directory named after the plugin ID. This directory should contain the plugin's dist files.")
	}

	if data.ID != "" && base != data.ID {
		pass.ReportResult(pass.AnalyzerName, noIdentRootDir, fmt.Sprintf("Archive should contain a directory named %s", data.ID), "The plugin archive file should contain a directory named after the plugin ID. This directory should contain the plugin's dist files.")
	} else {
		if noIdentRootDir.ReportAll {
			noIdentRootDir.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, noIdentRootDir, fmt.Sprintf("Archive contains directory named %s", data.ID), "")
		}
	}

	return nil, nil
}

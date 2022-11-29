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
	noIdentRootDir = &analysis.Rule{Name: "no-ident-root-dir"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "archivename",
	Requires: []*analysis.Analyzer{archive.Analyzer, metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{noIdentRootDir},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	base := filepath.Base(archiveDir)

	if base == "dist" {
		// Reporting here would be redundant, since they already know it's a
		// deprecated archive structure.
		if noIdentRootDir.ReportAll {
			noIdentRootDir.Severity = analysis.OK
			pass.Reportf(pass.AnalyzerName, noIdentRootDir, fmt.Sprintf("Archive contains directory named %s", data.ID))
		}
		return nil, nil
	}

	if data.ID != "" && base != data.ID {
		pass.Reportf(pass.AnalyzerName, noIdentRootDir, fmt.Sprintf("archive should contain a directory named %s", data.ID))
	} else {
		if noIdentRootDir.ReportAll {
			noIdentRootDir.Severity = analysis.OK
			pass.Reportf(pass.AnalyzerName, noIdentRootDir, fmt.Sprintf("Archive contains directory named %s", data.ID))
		}
	}

	return nil, nil
}

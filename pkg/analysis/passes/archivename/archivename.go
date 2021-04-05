package archivename

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var Analyzer = &analysis.Analyzer{
	Name:     "archivename",
	Requires: []*analysis.Analyzer{archive.Analyzer, metadata.Analyzer},
	Run:      run,
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
		return nil, nil
	}

	if data.ID != "" && base != data.ID {
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Error,
			Message:  fmt.Sprintf("archive should contain a directory named %s", data.ID),
		})
	}

	return nil, nil
}

package backenddebug

import (
	"encoding/json"
	"fmt"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"os"
	"path/filepath"
)

var (
	backendDebugFilePresent = &analysis.Rule{Name: "backend-debug-file-present", Severity: analysis.Error}
)

// Analyzer is an analyzer that checks if backend standalone debug files are included in the executable.
// If so, it reports an error, as the plugin can't be used properly in non-debug mode.
var Analyzer = &analysis.Analyzer{
	Name:     "backenddebug",
	Requires: []*analysis.Analyzer{archive.Analyzer, metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{backendDebugFilePresent},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}
	if data.Executable == "" {
		// Do not perform check for plugins without a backend executable
		return nil, nil
	}

	for _, fn := range []string{"standalone.txt", "pid.txt"} {
		if _, err := os.Stat(filepath.Join(archiveDir, fn)); err != nil {
			if os.IsNotExist(err) {
				// Banned file not found
				continue
			}
			return nil, fmt.Errorf("stat %q: %w", fn, err)
		}
		// Found a banned file
		pass.ReportResult(
			pass.AnalyzerName,
			backendDebugFilePresent,
			"found standalone backend file",
			fmt.Sprintf("You have bundled %q, which will make the plugin unusable in production mode. Please get rid of it", fn),
		)
	}
	return nil, nil
}

package sdkusage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"golang.org/x/mod/modfile"
)

var (
	goSdkNotUsed  = &analysis.Rule{Name: "go-sdk-not-used", Severity: analysis.Error}
	goModNotFound = &analysis.Rule{Name: "go-mod-not-found", Severity: analysis.Error}
)

// Analyzer is an analyzer that checks if backend standalone debug files are included in the executable.
// If so, it reports an error, as the plugin can't be used properly in non-debug mode.
var Analyzer = &analysis.Analyzer{
	Name:     "backenddebug",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer, metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{goSdkNotUsed, goModNotFound},
}

func run(pass *analysis.Pass) (interface{}, error) {

	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		// no source code found so we can't go.mod
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

	if !data.Backend && data.Executable == "" {
		// Do not perform check for plugins without a backend declared
		return nil, nil
	}

	goModPath := filepath.Join(sourceCodeDir, "go.mod")
	// check if go.mod exists
	if _, err := os.Stat(goModPath); err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			goModNotFound,
			"go.mod can not be found in your source code",
			"You have indicated your plugin uses a backend (backend=true), but go.mod can not be found in your source code. If your plugin has a backend component you must use go (golang)",
		)
		// go.mod not found
		return nil, nil
	}

	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			goModNotFound,
			"go.mod can not be read from your source code",
			"You have indicated your plugin uses a backend (backend=true), but go.mod can not be read from your source code. If your plugin has a backend component you must use go (golang)",
		)
		return nil, nil
	}

	goModParsed, err := modfile.Parse("go.mod", goModContent, nil)

	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			goModNotFound,
			"go.mod can not be parsed from your source code",
			"You have indicated your plugin uses a backend (backend=true), but go.mod can not be parsed from your source code. If your plugin has a backend component you must use go (golang)",
		)
		return nil, nil
	}
	hasGoSdk := false

	for _, req := range goModParsed.Require {
		if req.Mod.Path == "github.com/grafana/grafana-plugin-sdk-go" {
			hasGoSdk = true
		}
	}

	if !hasGoSdk {
		pass.ReportResult(
			pass.AnalyzerName,
			goSdkNotUsed,
			"Your plugin uses a backend (backend=true), but the Grafana go sdk is not used",
			"If your plugin has a backend component you must use Grafana go sdk (github.com/grafana/grafana-plugin-sdk-go)",
		)
		return nil, nil
	}

	return nil, nil
}

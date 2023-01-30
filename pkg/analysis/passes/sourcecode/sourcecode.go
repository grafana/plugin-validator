package sourcecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tailscale/hujson"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	sourceCodeNotProvided     = &analysis.Rule{Name: "source-code-not-provided", Severity: analysis.Warning}
	sourceCodeNotFound        = &analysis.Rule{Name: "source-code-not-found", Severity: analysis.Error}
	sourceCodeVersionMisMatch = &analysis.Rule{Name: "source-code-version-mismatch", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "sourcecode",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{sourceCodeNotFound, sourceCodeVersionMisMatch, sourceCodeNotProvided},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var metadata metadata.Metadata
	if err := json.Unmarshal(metadataBody, &metadata); err != nil {
		return nil, err
	}

	sourceCodeDir := pass.SourceCodeDir
	if sourceCodeDir == "" {
		// Silently skip check if no source code is provided, for backwards compatibility
		return nil, nil
	}

	packageJsonPath := filepath.Join(sourceCodeDir, "package.json")
	packageJson, err := parsePackageJson(packageJsonPath)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			sourceCodeNotFound,
			fmt.Sprintf("Could not find or parse package.json from %s", sourceCodeDir),
			"The package.json inside the provided source code can't be parsed or doesn't exist.",
		)
		return nil, nil
	}

	if packageJson.Version != metadata.Info.Version {
		pass.ReportResult(
			pass.AnalyzerName,
			sourceCodeVersionMisMatch,
			fmt.Sprintf("The version in package.json (%s) doesn't match the version in plugin.json (%s)", packageJson.Version, metadata.Info.Version),
			"The version in the source code package.json must match the version in plugin.json",
		)
		return nil, nil
	}

	return sourceCodeDir, nil
}

func parsePackageJson(packageJsonPath string) (*PackageJson, error) {
	rawPackageJson, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil, err
	}

	// using hujson first to allow some tolerance in the package.json
	// such as comments and trailing commas that nodejs allows
	stdPackageJson, err := hujson.Standardize(rawPackageJson)
	if err != nil {
		return &PackageJson{}, err
	}

	var packageJson PackageJson = PackageJson{}

	if err := json.Unmarshal(stdPackageJson, &packageJson); err != nil {
		return &PackageJson{}, err
	}
	return &packageJson, nil
}

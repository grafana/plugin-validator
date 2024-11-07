package packagejson

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tailscale/hujson"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
)

var (
	packagejsonNotFound        = &analysis.Rule{Name: "packagejson-not-found", Severity: analysis.Error}
	packageCodeVersionMisMatch = &analysis.Rule{Name: "packagecode-version-mismatch", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "packagejson",
	Requires: []*analysis.Analyzer{metadata.Analyzer, sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{packagejsonNotFound},
}

func run(pass *analysis.Pass) (interface{}, error) {

	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)

	// we don't fail published plugins for using toolkit (yet)
	if !ok || sourceCodeDir == "" {
		return nil, nil
	}

	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var md metadata.Metadata
	if err := json.Unmarshal(metadataBody, &md); err != nil {
		return nil, err
	}

	packageJsonPath := filepath.Join(sourceCodeDir, "package.json")
	parsedPackageJson, err := parsePackageJson(packageJsonPath)

	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			packagejsonNotFound,
			fmt.Sprintf("Could not find or parse package.json from %s", sourceCodeDir),
			"The package.json inside the provided source code can't be parsed or doesn't exist.",
		)
		return nil, nil
	}

	if parsedPackageJson.Version != md.Info.Version {
		pass.ReportResult(
			pass.AnalyzerName,
			packageCodeVersionMisMatch,
			fmt.Sprintf("The version in package.json (%s) doesn't match the version in plugin.json (%s)", parsedPackageJson.Version, md.Info.Version),
			"The version in the source code package.json must match the version in plugin.json",
		)
		return nil, nil
	}

	return parsedPackageJson, nil
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

	var packageJson PackageJson

	if err := json.Unmarshal(stdPackageJson, &packageJson); err != nil {
		return &PackageJson{}, err
	}
	return &packageJson, nil
}

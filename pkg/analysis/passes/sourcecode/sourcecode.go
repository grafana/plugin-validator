package sourcecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/archivetool"
	"github.com/grafana/plugin-validator/pkg/repotool"
)

var (
	sourceCodeNotProvided      = &analysis.Rule{Name: "source-code-not-provided", Severity: analysis.Warning}
	sourceCodeNotFound         = &analysis.Rule{Name: "source-code-not-found", Severity: analysis.Error}
	sourceCodeVersionMissMatch = &analysis.Rule{Name: "source-code-version-missmatch", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "sourcecode",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{sourceCodeNotFound, sourceCodeVersionMissMatch},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

	var metadata metadata.Metadata
	if err := json.Unmarshal(metadataBody, &metadata); err != nil {
		fmt.Println("error unmarshalling metadata")
		return nil, err
	}

	if pass.SourceCodeUri == "" {
		pass.ReportResult(pass.AnalyzerName, sourceCodeNotProvided, "source code not provided", "")
		return nil, nil
	}

	// TODO handle cleanup
	sourceCodeDir, _, err := getSourceCodeDir(pass.SourceCodeUri)
	fmt.Println("sourceCodeDir", pass.SourceCodeUri, sourceCodeDir)
	fmt.Println("sourceCodeDir", sourceCodeDir)
	if err != nil || sourceCodeDir == "" {
		pass.ReportResult(pass.AnalyzerName, sourceCodeNotFound, fmt.Sprintf("The provided URL %s does not point to a valid source code repository", pass.SourceCodeUri), "If you are passing a Git ref or sub-directory in the URL make sure they are correct.")
		return "", nil
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
			sourceCodeVersionMissMatch,
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
		return &PackageJson{}, err
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

func getSourceCodeDir(sourceCodeUri string) (string, func(), error) {
	// if sourceCodeUrl has a .zip extension
	if strings.HasPrefix(sourceCodeUri, "file://") {
		sourceCodeDir := strings.TrimPrefix(sourceCodeUri, "file://")
		if _, err := os.Stat(sourceCodeDir); err != nil {
			return "", nil, err
		}
		return sourceCodeDir, func() {}, nil
	}

	if strings.HasSuffix(sourceCodeUri, ".zip") {
		extractedDir, sourceCodeCleanUp, err := archivetool.ArchiveToLocalPath(sourceCodeUri)
		if err != nil {
			return "", sourceCodeCleanUp, fmt.Errorf("couldn't extract source code archive: %s. %w", sourceCodeUri, err)
		}
		return extractedDir, sourceCodeCleanUp, nil
	}

	extractedGitRepo, sourceCodeCleanUp, err := repotool.GitUrlToLocalPath(sourceCodeUri)
	if err != nil {
		return "", sourceCodeCleanUp, err
	}
	return extractedGitRepo, sourceCodeCleanUp, nil
}

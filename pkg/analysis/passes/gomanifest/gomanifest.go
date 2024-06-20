package gomanifest

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	noGoManifest      = &analysis.Rule{Name: "no-go-manifest", Severity: analysis.Error}
	invalidGoManifest = &analysis.Rule{Name: "invalid-go-manifest", Severity: analysis.Error}
	goManifestIssue   = &analysis.Rule{Name: "go-manifest-issue", Severity: analysis.Error}
)

type ManifestIssue struct {
	file string
	err  error
}

var Analyzer = &analysis.Analyzer{
	Name:     "go-manifest",
	Requires: []*analysis.Analyzer{archive.Analyzer, sourcecode.Analyzer, metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{noGoManifest, invalidGoManifest, goManifestIssue},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}
	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}
	if !data.Backend {
		// not a backend plugin so we don't need to check the manifest
		return nil, nil
	}

	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, errors.New("archive dir not found")
	}
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		// no source code found so we can't check the manifest
		return nil, nil
	}

	goFiles, err := doublestar.FilepathGlob(sourceCodeDir + "/**/*.go")
	if err != nil {
		return nil, nil
	}
	if len(goFiles) == 0 {
		// no go files found so we can't check the manifest
		return nil, nil
	}

	manifestFilePath := filepath.Join(archiveDir, "go_plugin_build_manifest")
	logme.DebugFln("manifestFilePath: %s", manifestFilePath)
	maniFestFiles, err := parseManifestFile(manifestFilePath)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			noGoManifest,
			"Could not find or parse Go manifest file",
			"Your source code contains Go files but there's no Go build manifest. Make sure you are using the latest version of the Go plugin SDK",
		)
		return nil, nil
	}

	issues, err := verifyManifest(maniFestFiles, goFiles, sourceCodeDir)
	// this is a rare error. most likely related to errors in the file system or corrupted files
	if err != nil {
		logme.DebugFln("verifyManifest error: %s", err)
		pass.ReportResult(
			pass.AnalyzerName,
			invalidGoManifest,
			"Error reading your go manifest file",
			"Your source code contains Go files but there's an issue with the go build manifest. Make sure you are using the latest version of the Go plugin SDK",
		)
		return nil, nil
	}

	for _, issue := range issues {
		pass.ReportResult(
			pass.AnalyzerName,
			goManifestIssue,
			fmt.Sprintf(
				"Invalid Go manifest file: %s",
				issue.file,
			),
			issue.err.Error(),
		)
	}

	return nil, nil

}

// parseManifestFile parses the manifest file and returns a map[string]string
// the content of the manifest file
func parseManifestFile(file string) (map[string]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	manifest := make(map[string]string)
	// read file line by line
	fileReader := bufio.NewScanner(f)
	fileReader.Split(bufio.ScanLines)
	for fileReader.Scan() {
		line := fileReader.Text()
		// skip empty lines
		if len(line) == 0 {
			continue
		}
		// fail if line does not contain a colon
		if !strings.Contains(line, ":") {
			return nil, fmt.Errorf("invalid line in manifest file: %s", line)
		}
		parsedLine := strings.Split(line, ":")
		// fail if line  is not in the format key:value
		if len(parsedLine) != 2 {
			return nil, fmt.Errorf("invalid line in manifest file: %s", line)
		}
		sha256sum := strings.TrimSpace(parsedLine[0])
		fileName := normalizeFileName(strings.TrimSpace(parsedLine[1]))
		// format the manifest fileName:sha256sum
		manifest[fileName] = sha256sum
	}

	if fileReader.Err() != nil {
		return nil, fileReader.Err()
	}

	return manifest, nil
}

func normalizeFileName(fileName string) string {
	// takes a filename that might have windows or linux separators and converts them to a linux separator
	return strings.Replace(fileName, "\\", "/", -1)
}

func verifyManifest(
	manifest map[string]string,
	goFiles []string,
	sourceCodeDir string,
) ([]ManifestIssue, error) {
	manifestIssues := []ManifestIssue{}

	for _, goFilePath := range goFiles {
		goFileRelativePath, err := filepath.Rel(sourceCodeDir, goFilePath)
		if err != nil {
			return nil, err
		}
		// calculate the linuxSha256sum of the go file
		linuxSha256sum, windowsSha256Sum, err := hashFileContent(goFilePath)
		if err != nil {
			return nil, err
		}
		// check if the sha256sum is in the manifest
		manifestSha256sum, ok := manifest[goFileRelativePath]

		if !ok {
			manifestIssues = append(manifestIssues, ManifestIssue{
				file: goFileRelativePath,
				err: fmt.Errorf(
					"file %s is in the source code but not in the manifest",
					goFileRelativePath,
				),
			})
			continue
		}
		// check if the sha256sum in the manifest matches the calculated sha256sum
		if linuxSha256sum != manifestSha256sum && windowsSha256Sum != manifestSha256sum {
			manifestIssues = append(manifestIssues, ManifestIssue{
				file: goFileRelativePath,
				err: fmt.Errorf(
					"sha256sum of %s (%s) does not match manifest",
					goFileRelativePath,
					linuxSha256sum,
				),
			})
			continue
		}
	}

	// find files in manifest that are not in the source code
	for fileName := range manifest {
		if _, err := os.Stat(filepath.Join(sourceCodeDir, fileName)); err != nil {
			manifestIssues = append(manifestIssues, ManifestIssue{
				file: fileName,
				err:  fmt.Errorf("%s is in the manifest but not in source code", fileName),
			})
		}
	}

	return manifestIssues, nil
}

// we hash the file content using the sha256 algorithm
// a second hash is created using the line endings that windows uses
// for plugins compiled with native windows that use windows line endings
func hashFileContent(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	// Normalize data to Linux line endings and calculate the hash
	linuxLineEndData := strings.ReplaceAll(string(data), "\r\n", "\n")
	hLinux := sha256.Sum256([]byte(linuxLineEndData))
	linuxHash := hex.EncodeToString(hLinux[:])

	// Normalize data to Windows line endings and calculate the hash
	windowsLineEndData := strings.ReplaceAll(string(linuxLineEndData), "\n", "\r\n")
	hWindows := sha256.Sum256([]byte(windowsLineEndData))
	windowsHash := hex.EncodeToString(hWindows[:])

	return linuxHash, windowsHash, nil
}

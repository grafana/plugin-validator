package gomanifest

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	noGoManifest      = &analysis.Rule{Name: "no-go-manifest", Severity: analysis.Warning}
	invalidGoManifest = &analysis.Rule{Name: "invalid-go-manifest", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "go-manifest",
	Requires: []*analysis.Analyzer{archive.Analyzer, sourcecode.Analyzer, metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{noGoManifest, invalidGoManifest},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, errors.New("metadata not found")
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
		pass.ReportResult(pass.AnalyzerName, noGoManifest,
			"Could not find or parse Go manifest file",
			"Your source code contains Go files but there's no Go build manifest. Make sure you are using the latest version of the Go plugin SDK")
		return nil, nil
	}

	err = verifyManifest(maniFestFiles, goFiles, sourceCodeDir)
	if err != nil {
		logme.DebugFln("verifyManifest error: %s", err)
		pass.ReportResult(pass.AnalyzerName, invalidGoManifest,
			"The Go build manifest does not match the source code",
			"The provided Go build manifest does not match the provided source code. If you are providing a git repository URL make sure to include the correct ref (branch or tag) in the URL and it includes all the Go files used to build the plugin binaries")
		return nil, nil
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

func verifyManifest(manifest map[string]string, goFiles []string, sourceCodeDir string) error {
	for _, goFilePath := range goFiles {
		goFileRelativePath, err := filepath.Rel(sourceCodeDir, goFilePath)
		if err != nil {
			return err
		}
		// calculate the sha256sum of the go file
		sha256sum, err := hashFileContent(goFilePath)
		if err != nil {
			return err
		}
		// check if the sha256sum is in the manifest
		manifestSha256sum, ok := manifest[goFileRelativePath]

		if !ok {
			return fmt.Errorf("could not find file %s with hash %s in manifest", goFileRelativePath, sha256sum)
		}
		// check if the sha256sum in the manifest matches the calculated sha256sum
		if sha256sum != manifestSha256sum {
			return fmt.Errorf("sha256sum of %s does not match manifest", goFilePath)
		}
	}

	// find files in manifest that are not in the source code
	for fileName := range manifest {
		if _, err := os.Stat(filepath.Join(sourceCodeDir, fileName)); err != nil {
			return fmt.Errorf("could not find %s in source code", fileName)
		}
	}

	return nil
}

func hashFileContent(path string) (string, error) {
	// Handle hashing big files.
	// Source: https://stackoverflow.com/q/60328216/1722542

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func() {
		err = f.Close()
		if err != nil {
			fmt.Printf("error closing file for hashing: %v", err)
		}
	}()

	buf := make([]byte, 1024*1024)
	h := sha256.New()

	for {
		bytesRead, err := f.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return "", err
			}
			_, err = h.Write(buf[:bytesRead])
			if err != nil {
				return "", err
			}
			break
		}
		_, err = h.Write(buf[:bytesRead])
		if err != nil {
			return "", err
		}
	}

	fileHash := hex.EncodeToString(h.Sum(nil))
	return fileHash, nil
}

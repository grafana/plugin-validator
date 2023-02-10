package gomanifest

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	noGoManifest      = &analysis.Rule{Name: "no-go-manifest", Severity: analysis.Warning}
	invalidGoManifest = &analysis.Rule{Name: "invalid-go-manifest", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "go-manifest",
	Requires: []*analysis.Analyzer{archive.Analyzer, sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{noGoManifest, invalidGoManifest},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, errors.New("archive dir not found")
	}
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		// no source code found so we can't check the manifest
		return nil, nil
	}

	goFiles := getGoFiles(sourceCodeDir)
	if len(goFiles) == 0 {
		// no go files found so we can't check the manifest
		return nil, nil
	}

	manifestFilePath := filepath.Join(archiveDir, "go_plugin_build_manifest")
	logme.DebugFln("manifestFilePath: %s", manifestFilePath)
	maniFestFiles, err := parseManifestFile(manifestFilePath)
	if err != nil {
		pass.ReportResult(pass.AnalyzerName, noGoManifest,
			"Could not find or parse go manifest file",
			"Your sourcecode contains go files but there's no go build manifest. Make sure you are using the latest version of the go plugin SDK")
		return nil, nil
	}

	err = verifyManifest(maniFestFiles, goFiles, sourceCodeDir)
	if err != nil {
		logme.DebugFln("verifyManifest error: %s", err)
		pass.ReportResult(pass.AnalyzerName, invalidGoManifest,
			"The go build manifest does not match the source code",
			"The provided go build manifest does not match the provided source code. If you are providing a git repository URL make sure to include the correct ref (branch or tag) in the URL and it includes all the go files used to build the plugin binaries")
		return nil, nil
	}

	return nil, nil

}

// return all go files in dir using filepath.WalkDir and with error handling
func getGoFiles(dir string) []string {
	goFiles := []string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".go" {
			goFiles = append(goFiles, path)
		}
		return nil
	})

	if err != nil {
		logme.Errorln(err)
	}

	return goFiles
}

// parseManifestFile parses the manifest file and returns a ManifestFile struct
// it does not verify the signature only returns the content
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
		// format the manifest fileName:sha256sum
		manifest[strings.TrimSpace(parsedLine[1])] = strings.TrimSpace(parsedLine[0])
	}

	return manifest, nil
}

func verifyManifest(manifest map[string]string, goFiles []string, sourceCodeDir string) error {
	for _, goFile := range goFiles {
		goFileRelativePath, err := filepath.Rel(sourceCodeDir, goFile)
		if err != nil {
			return err
		}
		// calculate the sha256sum of the go file
		sha256sum, err := calculateSha256sum(goFile)
		if err != nil {
			return err
		}
		// check if the sha256sum is in the manifest
		manifestSha256sum, ok := manifest[goFileRelativePath]
		if !ok {
			return fmt.Errorf("could not find %s in manifest", goFile)
		}
		// check if the sha256sum in the manifest matches the calculated sha256sum
		if sha256sum != manifestSha256sum {
			return fmt.Errorf("sha256sum of %s does not match manifest", goFile)
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

func calculateSha256sum(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return strings.ToLower(fmt.Sprintf("%x", h.Sum(nil))), nil
}

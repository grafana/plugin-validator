package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"

	// even though deprecated this is what grafana is using at the moment
	// https://github.com/grafana/grafana/blob/main/pkg/plugins/manager/signature/manifest.go
	"golang.org/x/crypto/openpgp/clearsign"
)

var (
	unsignedPlugin  = &analysis.Rule{Name: "unsigned-plugin", Severity: analysis.Warning}
	undeclaredFiles = &analysis.Rule{Name: "undeclared-files", Severity: analysis.Error}
	emptyManifest   = &analysis.Rule{Name: "empty-manifest", Severity: analysis.Error}
	wrongManifest   = &analysis.Rule{Name: "wrong-manifest", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "manifest",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{unsignedPlugin, undeclaredFiles, emptyManifest, wrongManifest},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	b, err := os.ReadFile(filepath.Join(archiveDir, "MANIFEST.txt"))
	if err != nil {
		pass.ReportResult(pass.AnalyzerName, unsignedPlugin, "unsigned plugin", "MANIFEST.txt file not found. Please refer to the documentation for how to sign a plugin. https://grafana.com/docs/grafana/latest/developers/plugins/sign-a-plugin/")
		return nil, nil
	}

	if len(b) == 0 {
		pass.ReportResult(pass.AnalyzerName, emptyManifest, "empty manifest", "MANIFEST.txt file is empty. Please refer to the documentation for how to sign a plugin. https://grafana.com/docs/grafana/latest/developers/plugins/sign-a-plugin/")
		return nil, nil
	}

	manifest, err := parseManifestFile(b)
	if err != nil {
		pass.ReportResult(pass.AnalyzerName, wrongManifest, "could not parse MANIFEST.txt", "MANIFEST.txt file is not a valid manifest. Please refer to the documentation for how to sign a plugin. https://grafana.com/docs/grafana/latest/developers/plugins/sign-a-plugin/")
		return nil, nil
	}

	if (manifest.Files == nil) || (len(manifest.Files) == 0) {
		pass.ReportResult(pass.AnalyzerName, undeclaredFiles, "no files declared in MANIFEST.txt", "No files declared in MANIFEST.txt")
		return nil, nil
	}

	// check if all existing files are declared in the manifest
	_ = filepath.Walk(archiveDir, func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if file.IsDir() {
			return nil
		}
		if file.Name() == "MANIFEST.txt" {
			return nil
		}
		// remove archiveDir from path
		relativePath := path[len(archiveDir)+1:]
		if _, ok := manifest.Files[relativePath]; !ok {
			pass.ReportResult(pass.AnalyzerName, undeclaredFiles, "undeclared files in MANIFEST", fmt.Sprintf("File %s is not declared in MANIFEST.txt", relativePath))
		}
		return nil
	})

	// check if all declared files exist
	for path := range manifest.Files {
		if _, err := os.Stat(filepath.Join(archiveDir, path)); os.IsNotExist(err) {
			pass.ReportResult(pass.AnalyzerName, undeclaredFiles, "declared files in MANIFEST not present", fmt.Sprintf("File %s is declared in MANIFEST.txt but does not exist", path))
		}
	}

	if unsignedPlugin.ReportAll {
		unsignedPlugin.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, unsignedPlugin, "MANIFEST.txt: plugin is signed", "")
	}

	return b, nil
}

// parseManifestFile parses the manifest file and returns a ManifestFile struct
// it does not verify the signature only returns the content
func parseManifestFile(b []byte) (ManifestFile, error) {
	block, _ := clearsign.Decode(b)
	manifestFile := ManifestFile{}
	content := (string(block.Plaintext))
	err := json.Unmarshal([]byte(content), &manifestFile)
	if err != nil {
		return manifestFile, err
	}
	return manifestFile, nil
}

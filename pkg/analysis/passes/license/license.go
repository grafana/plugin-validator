package license

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/go-enry/go-license-detector/v4/licensedb/filer"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	licenseNotProvided = &analysis.Rule{Name: "license-not-provided", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "license",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{licenseNotProvided},
}

var validLicenseStart = []string{"AGPL-3.0", "Apache-2.0", "MIT"}

const minRequiredConfidenceLevel float32 = 0.9

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	// validate that a LICENSE file is provided (go standard lib method)
	licenseFilePath := filepath.Join(archiveDir, "LICENSE")
	licenseFile, err := os.Stat(licenseFilePath)
	if err != nil || licenseFile.IsDir() {
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "LICENSE file not found", "Could not find a license file inside the plugin archive. Please make sure to include a LICENCE file in your archive.")
		return nil, nil
	}

	// validate that the LICENSE file is exists (filer lib method)
	filer, err := filer.FromDirectory(archiveDir)
	if err != nil {
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "LICENSE file not found", "Could not find a license file inside the plugin archive. Please make sure to include a LICENCE file in your archive.")
		return nil, nil
	}

	// validate that the LICENSE file is parseable (go-license-detector lib method)
	licenses, err := licensedb.Detect(filer)
	if err != nil {
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "LICENSE file could not be parsed.", "Could not parse the license file inside the plugin archive. Please make sure to include a valid license in your LICENSE file in your archive.")
		return nil, nil
	}

	var foundLicense = false
	for licenseName, licenseData := range licenses {
		if licenseData.Confidence >= minRequiredConfidenceLevel && isValidLicense(licenseName) {
			foundLicense = true
			break
		}
	}

	if !foundLicense {
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "Valid license not found", "The provided license is not compatible with Grafana plugins. Please refer to https://grafana.com/licensing/ for more information.")
	} else if licenseNotProvided.ReportAll {
		licenseNotProvided.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "License found", "Found a valid license file inside the plugin archive.")
	}

	return nil, nil
}

func isValidLicense(licenseName string) bool {
	for _, prefix := range validLicenseStart {
		if strings.HasPrefix(licenseName, prefix) {
			return true
		}
	}
	return false
}

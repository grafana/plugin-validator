package license

import (
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

var validLicenseStart = []string{"AGPL-3.0", "Apache-2.0"}
var minRequiredConfidenceLevel float32 = 0.9

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)

	filer, err := filer.FromDirectory(archiveDir)
	if err != nil {
		return nil, err
	}

	licenses, err := licensedb.Detect(filer)
	if err != nil {
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "License not found", "Could not find or parse the license file inside the plugin archive. Please make sure to include a LICENCE file in your archive.")
		return nil, nil
	}

	var foundLicense = false
	for licenseName, licenseData := range licenses {
		if licenseData.Confidence >= minRequiredConfidenceLevel {
			for _, prefix := range validLicenseStart {
				if strings.HasPrefix(licenseName, prefix) {
					foundLicense = true
				}
			}
		}
	}

	if !foundLicense {
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "Valid license not found", "Could not find a license file inside the plugin archive or the provided license is not compatible with Grafana plugins. Please refer to https://grafana.com/licensing/ for more information.")
	} else if licenseNotProvided.ReportAll {
		licenseNotProvided.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "License found", "Found a valid license file inside the plugin archive.")
	}

	return nil, nil
}

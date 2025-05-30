package license

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-enry/go-license-detector/v4/licensedb"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	licenseNotProvided     = &analysis.Rule{Name: "license-not-provided", Severity: analysis.Error}
	licenseNotValid        = &analysis.Rule{Name: "license-not-valid", Severity: analysis.Error}
	licenseWithGenericText = &analysis.Rule{
		Name:     "license-with-generic-text",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "license",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{licenseNotProvided, licenseNotValid, licenseWithGenericText},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "License Type",
		Description: "Checks the declared license is one of: BSD, MIT, Apache 2.0, LGPL3, GPL3, AGPL3.",
	},
}

// note: these follow the SPDX license list: https://spdx.org/licenses/
// go-license-detector uses the same list with the same upper/lower case
var validLicensesRegex = []*regexp.Regexp{
	regexp.MustCompile(`^0BSD$`),
	regexp.MustCompile(`^BSD-.*$`),
	regexp.MustCompile(`^MIT.*$`),
	regexp.MustCompile(`^Apache-2.0$`),
	regexp.MustCompile(`^LGPL-3.*$`),
	regexp.MustCompile(`^GPL-3.0.*$`),
	regexp.MustCompile(`^AGPL-3.0.*$`),
}

const minRequiredConfidenceLevel float32 = 0.9

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	// validate that a LICENSE file is provided (go standard lib method)
	licenseFilePath := filepath.Join(archiveDir, "LICENSE")
	licenseContents, err := os.ReadFile(licenseFilePath)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			licenseNotProvided,
			"LICENSE file not found",
			"Could not find a license file inside the plugin archive. Please make sure to include a LICENSE file in your archive.",
		)
		return nil, nil
	}

	// create a new temporal directory
	tmpDir, err := os.MkdirTemp(os.TempDir(), "plugin-validator")
	if err != nil {
		return nil, nil
	}
	defer os.RemoveAll(tmpDir)

	// copy the license file to the temporal directory
	err = os.WriteFile(filepath.Join(tmpDir, "LICENSE"), licenseContents, 0644)
	if err != nil {
		return nil, nil
	}

	result := licensedb.Analyse(tmpDir)
	if len(result) == 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			licenseNotProvided,
			"LICENSE file not found",
			"Could not find a license file inside the plugin archive. Please make sure to include a LICENSE file in your archive.",
		)
		return nil, nil
	}

	liceneseErr := result[0].ErrStr
	if liceneseErr != "" {
		pass.ReportResult(
			pass.AnalyzerName,
			licenseNotProvided,
			"LICENSE file could not be parsed.",
			"Could not parse the license file inside the plugin archive. Please make sure to include a valid license in your LICENSE file in your archive.",
		)
		return nil, nil
	}

	licenses := result[0].Matches

	var foundLicense = false
	for _, licenseData := range licenses {
		if licenseData.Confidence >= minRequiredConfidenceLevel &&
			isValidLicense(licenseData.License) {
			foundLicense = true
			break
		}
	}

	if !foundLicense {
		pass.ReportResult(
			pass.AnalyzerName,
			licenseNotValid,
			"Valid license not found",
			"The provided license is not compatible with Grafana plugins. Please refer to https://grafana.com/licensing/ for more information.",
		)
		return nil, nil

	} else if licenseNotProvided.ReportAll {
		licenseNotProvided.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "License found", "Found a valid license file inside the plugin archive.")
	}

	licenseContentStr := string(licenseContents)
	if strings.Contains(licenseContentStr, "{name of copyright owner}") ||
		strings.Contains(licenseContentStr, "{yyyy}") {
		pass.ReportResult(
			pass.AnalyzerName,
			licenseWithGenericText,
			"License file contains generic text",
			"Your current license file contains generic text from the license template. Please make sure to replace {name of copyright owner} and {yyyy} with the correct values in your LICENSE file.",
		)
	}

	return nil, nil
}

func isValidLicense(licenseName string) bool {
	for _, prefix := range validLicensesRegex {
		if prefix.MatchString(licenseName) {
			return true
		}
	}
	return false
}

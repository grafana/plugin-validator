package license

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/go-enry/go-license-detector/v4/licensedb/api"
	"github.com/go-enry/go-license-detector/v4/licensedb/filer"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	licenseNotProvided      = &analysis.Rule{Name: "license-not-provided", Severity: analysis.Error}
	licenseNotValid         = &analysis.Rule{Name: "license-not-valid", Severity: analysis.Error}
	licenseDetectionTimeout = &analysis.Rule{Name: "license-detection-timeout", Severity: analysis.Error}
	licenseWithGenericText  = &analysis.Rule{
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
	licenseFile, err := os.Stat(licenseFilePath)
	if err != nil || licenseFile.IsDir() {
		pass.ReportResult(
			pass.AnalyzerName,
			licenseNotProvided,
			"LICENSE file not found",
			"Could not find a license file inside the plugin archive. Please make sure to include a LICENSE file in your archive.",
		)
		return nil, nil
	}

	// validate that the LICENSE file exists (filer lib method)
	f, err := filer.FromDirectory(archiveDir)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			licenseNotProvided,
			"LICENSE file not found",
			"Could not find a license file inside the plugin archive. Please make sure to include a LICENSE file in your archive.",
		)
		return nil, nil
	}

	ch := make(chan map[string]api.Match, 1)
	defer close(ch)
	go func() {
		// validate that the LICENSE file is parseable (go-license-detector lib method)
		licenses, err := licensedb.Detect(f)
		if err != nil {
			ch <- nil
			return
		}
		ch <- licenses
	}()

	select {
	case licenses := <-ch:
		var foundLicense = false
		for licenseName, licenseData := range licenses {
			if licenseData.Confidence >= minRequiredConfidenceLevel && isValidLicense(licenseName) {
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
		} else if licenseNotProvided.ReportAll {
			licenseNotProvided.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, licenseNotProvided, "License found", "Found a valid license file inside the plugin archive.")
		}

		licenseContent, err := os.ReadFile(licenseFilePath)
		if err != nil {
			logme.Debugln("Could not read LICENSE file", err)
			return nil, nil
		}

		licenseContentStr := string(licenseContent)
		if strings.Contains(licenseContentStr, "{name of copyright owner}") ||
			strings.Contains(licenseContentStr, "{yyyy}") {
			pass.ReportResult(
				pass.AnalyzerName,
				licenseWithGenericText,
				"License file contains generic text",
				"Your current license file contains generic text from the license template. Please make sure to replace {name of copyright owner} and {yyyy} with the correct values in your LICENSE file.",
			)
		}
	case <-time.After(time.Second * 30):
		pass.ReportResult(
			pass.AnalyzerName,
			licenseDetectionTimeout,
			"LICENSE file detection timeout.",
			"Could not detect the license file inside the plugin archive within 30s. Please make sure to include a valid license in your LICENSE file in your archive.",
		)
		return nil, nil
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

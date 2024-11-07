package checksum

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	checksumInvalid     = &analysis.Rule{Name: "checksum-invalid", Severity: analysis.Error}
	checksumFormatError = &analysis.Rule{Name: "checksum-format-error", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "checksum",
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{checksumInvalid},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Checksum",
		Description:  "Validates that the passed checksum (as a validator arg) is the one calculated from the archive file.",
		Dependencies: "`checksum`",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {

	checksum := pass.CheckParams.Checksum

	// skip if no provided
	if checksum == "" || len(checksum) == 0 {
		return nil, nil
	}

	checksum, err := resolveCheckSum(checksum)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			checksumFormatError,
			fmt.Sprintf("%s: %s", err.Error(), pass.CheckParams.Checksum),
			"Make sure you provide a valid checksum as a string or a valid URL to a checksum. Only MD5 and SHA1 checksums are supported.",
		)
		return nil, nil
	}

	checkSumType := ""

	// determine if it is md5 or sha1
	if len(checksum) == 32 {
		checkSumType = "md5"
	} else if len(checksum) == 40 {
		checkSumType = "sha1"
	} else {
		pass.ReportResult(
			pass.AnalyzerName,
			checksumFormatError,
			fmt.Sprintf("Invalid checksum format: %s", checksum),
			"Make sure you provide a valid checksum as a string or a valid URL to a checksum. Only MD5 and SHA1 checksums are supported.",
		)
		return nil, nil
	}
	isError := false

	switch checkSumType {
	case "md5":
		if checksum != strings.ToLower(pass.CheckParams.ArchiveCalculatedMD5) {
			isError = true
		}
	case "sha1":
		if checksum != strings.ToLower(pass.CheckParams.ArchiveCalculatedSHA1) {
			isError = true
		}
	}

	if isError {
		pass.ReportResult(
			pass.AnalyzerName,
			checksumInvalid,
			fmt.Sprintf("The provided checksum %s does not match the plugin archive", checksum),
			"The plugin archive does not match the provided checksum in the submission form. Please double check you are providing the correct plugin archive and checksum.",
		)
	}

	return nil, nil
}

// sums can come as:
// 04b2f189616b65985d219ca094f3609b  arduino-cli.yaml
// 0b41e5bffba89764a494f420347fe5110fa2886c  arduino-cli.yaml
// 0b41e5bffba89764a494f420347fe5110fa2886c
// 04b2f189616b65985d219ca094f3609b
// we only care about the first part
func sanitizeCheckSum(checksum string) string {
	return strings.ToLower(strings.Fields(checksum)[0])
}

// checksum can be urls, we need to download
func resolveCheckSum(checksum string) (string, error) {

	checksum = strings.TrimSpace(checksum)

	if strings.HasPrefix(checksum, "https://") || strings.HasPrefix(checksum, "http://") {
		finalChecksum, err := getFirstLineFromURL(checksum)
		if err != nil {
			return "", err
		}
		if len(finalChecksum) == 0 {
			return "", errors.New(
				"Checksum URL is invalid. Please provide a URL with a direct download to the checksum.",
			)
		}
		return sanitizeCheckSum(finalChecksum), nil
	}
	return sanitizeCheckSum(checksum), nil
}

const (
	maxSize = 500 * 1024 // 500KB
	timeout = 30 * time.Second
)

func getFirstLineFromURL(url string) (string, error) {
	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		logme.DebugFln("Error reading body: %s", err.Error())
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("checksum file not found")
	}

	if resp.ContentLength > maxSize {
		return "", errors.New("checksum file is too large")
	}

	lr := &io.LimitedReader{R: resp.Body, N: maxSize}
	scanner := bufio.NewScanner(lr)
	scanner.Scan() // get the first line
	line := scanner.Text()

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return line, nil
}

package provenance

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	noProvenanceAttestation = &analysis.Rule{
		Name:     "no-provenance-attestation",
		Severity: analysis.Warning,
	}
	invalidProvenanceAttestation = &analysis.Rule{
		Name:     "invalid-provenance-attestation",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "provenance",
	Requires: []*analysis.Analyzer{},
	Run:      run,
	Rules: []*analysis.Rule{
		noProvenanceAttestation,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Provenance attestation validation",
		Description: "Validates the provenance attestation if the plugin was built with a pipeline supporting provenance attestation (e.g Github Actions).",
	},
}

var githubRe = regexp.MustCompile(`https://github\.com\/([^/]+)/([^/]+)`)
var githubToken = os.Getenv("GITHUB_TOKEN")

func run(pass *analysis.Pass) (interface{}, error) {

	if githubToken == "" {
		logme.Debugln(
			"Skipping provenance attestation check because GITHUB_TOKEN is not set",
		)
		return nil, nil
	}

	matches := githubRe.FindStringSubmatch(pass.CheckParams.SourceCodeReference)
	if matches == nil || len(matches) < 3 {
		detail := "Cannot verify plugin build. It is recommended to use a pipeline that supports provenance attestation, such as GitHub Actions. https://github.com/grafana/plugin-actions/tree/main/build-plugin"

		// add instructions if the source code reference is a github repo
		if strings.Contains(pass.CheckParams.ArchiveFile, "github.com") {
			detail = "Cannot verify plugin build. To enable verification, see the documentation on implementing build attestation: https://grafana.com/developers/plugin-tools/publish-a-plugin/build-automation#enable-provenance-attestation"
		}
		pass.ReportResult(
			pass.AnalyzerName,
			noProvenanceAttestation,
			"No provenance attestation. This plugin was built without build verification",
			detail,
		)
		return nil, nil
	}

	owner := matches[1]

	hasGithubProvenanceAttestationPipeline, err := hasGithubProvenanceAttestationPipeline(
		pass.CheckParams.ArchiveFile,
		owner,
	)
	if err != nil || !hasGithubProvenanceAttestationPipeline {
		message := "Cannot verify plugin build provenance attestation."
		pass.ReportResult(
			pass.AnalyzerName,
			invalidProvenanceAttestation,
			message,
			"Please verify your workflow attestation settings. See the documentation on implementing build attestation: https://grafana.com/developers/plugin-tools/publish-a-plugin/build-automation#enable-provenance-attestation",
		)
		return nil, nil
	}

	if noProvenanceAttestation.ReportAll {
		noProvenanceAttestation.Severity = analysis.OK
		pass.ReportResult(
			pass.AnalyzerName,
			noProvenanceAttestation,
			"Provenance attestation found",
			"", // TODO add details
		)
	}

	return nil, nil
}

func hasGithubProvenanceAttestationPipeline(assetPath string, owner string) (bool, error) {
	sha256sum, err := getFileSha256(assetPath)
	if err != nil {
		return false, err
	}

	url := fmt.Sprintf("https://api.github.com/users/%s/attestations/sha256:%s", owner, sha256sum)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", githubToken))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// A 200 status code means attestations were found
	if resp.StatusCode == http.StatusOK {
		logme.Debugln("Provenance attestation found. Got a 200 status code")
		return true, nil
	}

	// A 404 means no attestations were found
	if resp.StatusCode == http.StatusNotFound {
		logme.Debugln("Provenance attestation not found. Got a 404 status code")
		return false, nil
	}

	// Any other status code is treated as an error
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf(
		"unexpected response from GitHub API (status %d): %s",
		resp.StatusCode,
		string(body),
	)
}

func getFileSha256(assetPath string) (string, error) {
	// Check if file exists and is a regular file
	fileInfo, err := os.Stat(assetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", assetPath)
		}
		return "", fmt.Errorf("error accessing file: %w", err)
	}

	if !fileInfo.Mode().IsRegular() {
		return "", fmt.Errorf("file is not a regular file: %s", assetPath)
	}

	// Open and read the file
	file, err := os.Open(assetPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Calculate SHA256 hash
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Convert hash to hex string
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

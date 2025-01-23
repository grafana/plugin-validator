package provenance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func run(pass *analysis.Pass) (interface{}, error) {

	_, err := getGithubCliPath()
	if err != nil {
		logme.Debugln(
			"Skipping provenance attestation check because gh is not installed. Please install it and try again.",
			err,
		)
		return nil, nil
	}

	matches := githubRe.FindStringSubmatch(pass.CheckParams.SourceCodeReference)
	if matches == nil || len(matches) < 3 {
		detail := "Cannot verify plugin build. It is recommended to use a pipeline that supports provenance attestation, such as GitHub Actions. https://github.com/grafana/plugin-actions/tree/main/build-plugin"

		// add instructions if the source code reference is a github repo
		if strings.Contains(pass.CheckParams.ArchiveFile, "github.com") {
			detail = "Cannot verify plugin build. To enable verification, see the documentation on implementing build attestation: https://github.com/grafana/plugin-actions/tree/main/build-plugin#add-attestation-to-your-existing-workflow"
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
	repo := matches[2]

	attestationPipeline, err := getGithubProvenanceAttestationPipeline(
		pass.CheckParams.ArchiveFile,
		fmt.Sprintf("%s/%s", owner, repo),
	)
	if err != nil || attestationPipeline == "" {
		message := "Cannot verify plugin build."
		if err != nil {
			message = fmt.Sprintf("%s %s", message, err)
		}
		pass.ReportResult(
			pass.AnalyzerName,
			invalidProvenanceAttestation,
			message,
			"Please verify your workflow attestation settings. See the documentation on implementing build attestation: https://github.com/grafana/plugin-actions/tree/main/build-plugin#add-attestation-to-your-existing-workflow",
		)
		return nil, nil
	}

	if noProvenanceAttestation.ReportAll {
		noProvenanceAttestation.Severity = analysis.OK
		pass.ReportResult(
			pass.AnalyzerName,
			noProvenanceAttestation,
			"Provenance attestation found",
			attestationPipeline,
		)
	}

	return nil, nil
}

func getGithubProvenanceAttestationPipeline(assetPath string, repo string) (string, error) {

	ghCliBin, err := getGithubCliPath()
	if err != nil {
		return "", err
	}

	ghCommand := fmt.Sprintf(
		"%s attestation verify --format json --repo %s %s",
		ghCliBin,
		repo,
		assetPath,
	)

	//run ghCommand adding GH_TOKEN in a fresh ENV
	cmd := exec.Command("sh", "-c", ghCommand)
	cmd.Env = os.Environ()

	cmd.Stderr = &bytes.Buffer{}
	out, err := cmd.Output()
	if err != nil {
		errOut := ""
		if stderr, ok := cmd.Stderr.(*bytes.Buffer); ok {
			errOut = stderr.String()
			if strings.Contains(strings.ToLower(errOut), "http 404") {
				return "", fmt.Errorf("Attestation not found for asset %s", assetPath)
			}
		}
		return "", fmt.Errorf("Error running gh command: %s\n%s", err, errOut)
	}

	parsedResponse := []AttestationResponse{}

	err = json.Unmarshal(out, &parsedResponse)
	if err != nil {
		logme.Debugln("Error parsing gh command output: ", err)
		return "", err
	}

	if len(parsedResponse) == 0 {
		return "", fmt.Errorf("No provenance attestation returned by Github")
	}

	attestation := parsedResponse[0]

	if attestation.VerificationResult.Statement.Predicate.RunDetails.Metadata.InvocationID == "" {
		return "", fmt.Errorf("No provenance attestation InvocationID returned by Github")
	}

	return attestation.VerificationResult.Statement.Predicate.RunDetails.Metadata.InvocationID, nil
}

func getGithubCliPath() (string, error) {
	// currently the validator can only validate github actions provenance attestations
	if os.Getenv("GITHUB_TOKEN") == "" {
		return "", fmt.Errorf("GITHUB_TOKEN is not set")
	}

	var ghCliBin, err = exec.LookPath("gh")
	if err != nil || ghCliBin == "" {
		return "", fmt.Errorf("gh is not installed. Please install it and try again.")
	}

	return ghCliBin, nil

}

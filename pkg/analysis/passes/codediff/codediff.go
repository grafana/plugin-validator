package codediff

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/backendbinary"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/binarypermissions"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/coderules"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/gomanifest"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/jssourcemap"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/llmreview"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/manifest"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadatavalid"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/osvscanner"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/safelinks"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/trackingscripts"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/unsafesvg"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/virusscan"
	"github.com/grafana/plugin-validator/pkg/llmclient"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/versioncommitfinder"
)

//go:embed prompt.txt
var promptTemplate string

type LLMAnalysisResponse struct {
	Question     string   `json:"question"`
	Answer       string   `json:"answer"`
	RelatedFiles []string `json:"related_files"`
	CodeSnippet  string   `json:"code_snippet"`
	ShortAnswer  string   `json:"short_answer"`
}

var (
	codeDiffAnalysis = &analysis.Rule{
		Name:     "code-diff-analysis",
		Severity: analysis.SuspectedProblem,
	}
	codeDiffversions = &analysis.Rule{
		Name:     "code-diff-versions",
		Severity: analysis.SuspectedProblem,
	}
	codeDiffSkipped = &analysis.Rule{
		Name:     "code-diff-skipped",
		Severity: analysis.SuspectedProblem,
	}
)

// blockingAnalyzers contains validators that, if they report errors, should cause
// the code diff analysis to be skipped to save costs. These are grouped into:
// - Tier 1 (Structure): archive, metadata, metadatavalid, modulejs
// - Tier 2 (Security): coderules, trackingscripts, virusscan, safelinks, unsafesvg, osvscanner
// - Tier 3 (Integrity): gomanifest, binarypermissions, backendbinary, manifest
var blockingAnalyzers = []*analysis.Analyzer{
	// Tier 1: Fundamental structure issues
	archive.Analyzer,
	metadata.Analyzer,
	metadatavalid.Analyzer,
	modulejs.Analyzer,
	// Tier 2: Security/Policy violations
	coderules.Analyzer,
	trackingscripts.Analyzer,
	virusscan.Analyzer,
	safelinks.Analyzer,
	unsafesvg.Analyzer,
	osvscanner.Analyzer,
	// Tier 3: Build/Integrity issues
	gomanifest.Analyzer,
	jssourcemap.Analyzer,
	binarypermissions.Analyzer,
	backendbinary.Analyzer,
	manifest.Analyzer,
}

var Analyzer = &analysis.Analyzer{
	Name:     "codediff",
	Requires: append([]*analysis.Analyzer{llmreview.Analyzer}, blockingAnalyzers...),
	Run:      run,
	Rules:    []*analysis.Rule{codeDiffAnalysis, codeDiffversions, codeDiffSkipped},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Code Diff",
		Description:  "",
		Dependencies: "Google API Key with Generative AI access",
	},
}

var llmClient llmclient.LLMClient

func SetLLMClient(client llmclient.LLMClient) {
	llmClient = client
}

func init() {
	llmClient = llmclient.NewGeminiClient()
}

func isGitHubURL(url string) bool {
	return strings.Contains(strings.ToLower(url), "github.com")
}

func run(pass *analysis.Pass) (any, error) {
	// Check if any blocking analyzers reported errors - skip code diff to save costs
	// keep here before source code and key check for tests to work
	for _, analyzer := range blockingAnalyzers {
		if pass.AnalyzerHasErrors(analyzer) {
			pass.ReportResult(
				pass.AnalyzerName,
				codeDiffSkipped,
				fmt.Sprintf("Code diff skipped due to errors in %s", analyzer.Name),
				fmt.Sprintf(
					"Fix the errors reported by %s before code diff can run.",
					analyzer.Name,
				),
			)
			return nil, nil
		}
	}

	if pass.CheckParams.SourceCodeReference == "" {
		return nil, nil
	}

	if os.Getenv("SKIP_LLM_CODEDIFF") != "" {
		return nil, nil
	}

	if err := llmClient.CanUseLLM(); err != nil {
		return nil, nil
	}

	// Only support GitHub URLs for diff generation
	if !isGitHubURL(pass.CheckParams.SourceCodeReference) {
		logme.Debugln(
			"Source code reference is not a GitHub URL:",
			pass.CheckParams.SourceCodeReference,
		)
		return nil, nil
	}

	versions, cleanup, err := versioncommitfinder.FindPluginVersionsRefs(
		pass.CheckParams.SourceCodeReference,
		"",
	)
	if err != nil {
		logme.Debugln("Failed to find versions", err)
		return nil, nil
	}
	defer cleanup()

	// Generate and report diff links if both versions have commit SHAs
	if versions.CurrentGrafanaVersion != nil && versions.SubmittedGitHubVersion != nil &&
		versions.CurrentGrafanaVersion.CommitSHA != "" && versions.SubmittedGitHubVersion.CommitSHA != "" {

		// Generate GitHub compare URL
		diffURL := fmt.Sprintf(
			"https://github.com/%s/%s/compare/%s...%s",
			versions.Repository.Owner,
			versions.Repository.Repo,
			versions.CurrentGrafanaVersion.CommitSHA,
			versions.SubmittedGitHubVersion.CommitSHA,
		)

		logme.Debugln("Generated diff URL:", diffURL)

		// Report with clickable link
		message := fmt.Sprintf(
			"Code changes between versions %s → %s",
			versions.CurrentGrafanaVersion.Version,
			versions.SubmittedGitHubVersion.Version,
		)
		detail := fmt.Sprintf(
			"View code differences: [%s](%s)",
			diffURL,
			diffURL,
		)

		pass.ReportResult(pass.AnalyzerName, codeDiffversions, message, detail)

		// Run LLM analysis
		responses, err := runLLMAnalysis(
			versions.SubmittedGitHubVersion.Version,
			versions.SubmittedGitHubVersion.CommitSHA,
			versions.CurrentGrafanaVersion.Version,
			versions.CurrentGrafanaVersion.CommitSHA,
			versions.RepositoryPath,
		)
		if err != nil {
			logme.Debugln("Failed to run LLM analysis:", err)
			return nil, nil
		}

		// Report analysis results based on LLM responses
		for _, response := range responses {
			logme.Debugln("LLM response:", response.Question, response.Answer)
			if strings.ToLower(response.ShortAnswer) == "yes" {
				var detailParts []string

				detailParts = append(detailParts, response.Answer)

				if response.CodeSnippet != "" {
					detailParts = append(
						detailParts,
						fmt.Sprintf("**Code Snippet:**\n```\n%s\n```", response.CodeSnippet),
					)
				}

				if len(response.RelatedFiles) > 0 {
					detailParts = append(
						detailParts,
						fmt.Sprintf("**Files:** %s", strings.Join(response.RelatedFiles, ", ")),
					)
				}

				detail := strings.Join(detailParts, "\n\n")

				pass.ReportResult(
					pass.AnalyzerName,
					codeDiffAnalysis,
					fmt.Sprintf("Code Diff LLM flagged: %s", response.Question),
					detail,
				)
			}
		}

	} else {
		logme.Debugln("Cannot generate diff URL - missing commit SHAs or version information")
		if versions.CurrentGrafanaVersion == nil {
			logme.Debugln("Current Grafana version is nil")
		}
		if versions.SubmittedGitHubVersion == nil {
			logme.Debugln("Submitted GitHub version is nil")
		}
	}

	return nil, nil
}

func generatePrompt(newVersion, newCommit, currentVersion, currentCommit string) (string, error) {
	if newVersion == "" {
		return "", errors.New("new version is empty")
	}
	if newCommit == "" {
		return "", errors.New("new commit is empty")
	}
	if currentVersion == "" {
		return "", errors.New("current version is empty")
	}
	if currentCommit == "" {
		return "", errors.New("current commit is empty")
	}

	// Build questions section from llmreview questions
	var questionsSection strings.Builder
	for _, q := range llmreview.Questions {
		questionsSection.WriteString("* ")
		questionsSection.WriteString(q.Question)
		questionsSection.WriteString("\n")
	}

	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}

	data := map[string]any{
		"NewVersion":     newVersion,
		"NewCommit":      newCommit,
		"CurrentVersion": currentVersion,
		"CurrentCommit":  currentCommit,
		"Questions":      strings.TrimSuffix(questionsSection.String(), "\n"),
		"QuestionCount":  len(llmreview.Questions),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template: %w", err)
	}

	return buf.String(), nil
}

func runLLMAnalysis(
	newVersion, newCommit, currentVersion, currentCommit, repositoryPath string,
) ([]LLMAnalysisResponse, error) {
	// Generate the prompt with dynamic version/commit information
	prompt, err := generatePrompt(newVersion, newCommit, currentVersion, currentCommit)
	if err != nil {
		logme.Debugln("Failed to generate prompt:", err)
		return nil, err
	}

	llmclient.CleanUpPromptFiles(repositoryPath)

	// Call the LLM
	if err := llmClient.CallLLM(prompt, repositoryPath, nil); err != nil {
		logme.Debugln("Failed to call LLM:", err)
		return nil, err
	}

	// Read and parse the responses
	responsesPath := filepath.Join(repositoryPath, "replies.json")
	if _, err := os.Stat(responsesPath); err != nil {
		logme.Debugln("replies.json file not found:", err)
		return nil, fmt.Errorf("replies.json file not found: %w", err)
	}

	responsesData, err := os.ReadFile(responsesPath)
	if err != nil {
		logme.Debugln("Failed to read replies.json:", err)
		return nil, fmt.Errorf("failed to read replies.json: %w", err)
	}

	var responses []LLMAnalysisResponse
	if err := json.Unmarshal(responsesData, &responses); err != nil {
		logme.Debugln("Failed to parse replies.json:", err)
		return nil, fmt.Errorf("failed to parse replies.json: %w", err)
	}

	logme.Debugln("LLM analysis completed successfully")
	return responses, nil
}

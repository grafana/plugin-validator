package codediff

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/llmreview"
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
)

var Analyzer = &analysis.Analyzer{
	Name:     "codediff",
	Requires: []*analysis.Analyzer{llmreview.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{codeDiffAnalysis, codeDiffversions},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Code Diff",
		Description:  "",
		Dependencies: "Google API Key with Generative AI access",
	},
}
var geminiKey = os.Getenv("GEMINI_API_KEY")

func isGitHubURL(url string) bool {
	return strings.Contains(strings.ToLower(url), "github.com")
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.CheckParams.SourceCodeReference == "" {
		return nil, nil
	}

	if geminiKey == "" {
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
			"View code differences: %s",
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
				pass.ReportResult(
					pass.AnalyzerName,
					codeDiffAnalysis,
					response.Question,
					response.Answer,
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

func isNpxAvailable() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

func callLLM(prompt, repositoryPath string) error {
	if !isNpxAvailable() {
		logme.Debugln("npx is not available in PATH")
		return errors.New("npx is not available in PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"npx",
		// do not confirm install
		"-y",
		"https://github.com/google-gemini/gemini-cli",
		// gemini "YOLO" mode so it can use al the tools
		"-y",
	)
	cmd.Dir = repositoryPath
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logme.Debugln("Running gemini CLI analysis in directory:", repositoryPath)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logme.Debugln("Gemini CLI timed out after 5 minutes")
		} else {
			logme.Debugln("Gemini CLI failed:", err)
		}
	}

	return nil

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

	// clean up files from repositoryPath
	cleanFiles := []string{"replies.json", ".nvmrc", "GEMINI.MD"}
	for _, file := range cleanFiles {
		filePath := filepath.Join(repositoryPath, file)
		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err != nil {
				logme.Debugln("Failed to remove file:", err)
			}
		}
	}

	// Call the LLM
	if err := callLLM(prompt, repositoryPath); err != nil {
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

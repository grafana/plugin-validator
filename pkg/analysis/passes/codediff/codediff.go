package codediff

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
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
	"github.com/grafana/plugin-validator/pkg/llmconfig"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/versioncommitfinder"
)

//go:embed prompt.txt
var promptTemplate string

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
		Dependencies: "API key for one of: Anthropic (ANTHROPIC_API_KEY), OpenAI (OPENAI_API_KEY), or Google (GEMINI_API_KEY)",
	},
}

var (
	agenticClient llmclient.AgenticClient
)

func SetAgenticClient(client llmclient.AgenticClient) {
	agenticClient = client
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

	llmCfg := llmconfig.Resolve()
	if llmCfg == nil {
		logme.Debugln("Skipping LLM code diff analysis: no API key set (ANTHROPIC_API_KEY, OPENAI_API_KEY, or GEMINI_API_KEY)")
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
		answers, err := runLLMAnalysis(
			llmCfg,
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

		// Report analysis results: flag answers that don't match expected.
		// Answers with Error set (e.g. budget exhausted) are skipped.
		for i, answer := range answers {
			if answer.Error != "" {
				logme.Debugln("LLM response error for question:", answer.Question, answer.Error)
				continue
			}
			logme.Debugln("LLM response:", answer.Question, answer.Answer)

			if answer.ShortAnswer != llmreview.Questions[i].ExpectedAnswer {
				var detailParts []string

				detailParts = append(detailParts, answer.Answer)

				if answer.CodeSnippet != "" {
					detailParts = append(
						detailParts,
						fmt.Sprintf("**Code Snippet:**\n```\n%s\n```", answer.CodeSnippet),
					)
				}

				if len(answer.Files) > 0 {
					detailParts = append(
						detailParts,
						fmt.Sprintf("**Files:** %s", strings.Join(answer.Files, ", ")),
					)
				}

				detail := strings.Join(detailParts, "\n\n")

				pass.ReportResult(
					pass.AnalyzerName,
					codeDiffAnalysis,
					fmt.Sprintf("Code Diff LLM flagged: %s", answer.Question),
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

func generateSystemPrompt(
	newVersion, newCommit, currentVersion, currentCommit string,
) (string, error) {
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

	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}

	data := map[string]any{
		"NewVersion":     newVersion,
		"NewCommit":      newCommit,
		"CurrentVersion": currentVersion,
		"CurrentCommit":  currentCommit,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template: %w", err)
	}

	return buf.String(), nil
}

// buildQuestionWithContext wraps a question with a reminder about the diff context
// so the agent doesn't lose track of what it's comparing across sequential questions.
func buildQuestionWithContext(question, currentCommit, newCommit string) string {
	return fmt.Sprintf(
		"You are comparing changes between commits %s and %s. Feel free to use git diff and git show to understand what changed. If a change looks significant, compare full file versions to understand context. %s.\n First verify in which current git ref you are",
		currentCommit,
		newCommit,
		question,
	)
}

func runLLMAnalysis(
	llmCfg *llmconfig.ProviderConfig,
	newVersion, newCommit, currentVersion, currentCommit, repositoryPath string,
) ([]llmclient.AnswerSchema, error) {
	systemPrompt, err := generateSystemPrompt(newVersion, newCommit, currentVersion, currentCommit)
	if err != nil {
		logme.Debugln("Failed to generate system prompt:", err)
		return nil, err
	}

	llmclient.CleanUpPromptFiles(repositoryPath)

	// Build questions with diff context reminder
	questions := make([]string, len(llmreview.Questions))
	for i, q := range llmreview.Questions {
		questions[i] = buildQuestionWithContext(q.Question, currentCommit, newCommit)
	}

	// Use mock client if set (tests), otherwise create a real one
	client := agenticClient
	if client == nil {
		client, err = llmclient.NewAgenticClient(&llmclient.AgenticCallOptions{
			Provider:     llmCfg.Provider,
			Model:        llmCfg.Model,
			APIKey:       llmCfg.APIKey,
			SystemPrompt: systemPrompt,
		})
		if err != nil {
			logme.Debugln("Failed to create agentic client:", err)
			return nil, err
		}
	}

	answers, err := client.CallLLM(context.Background(), questions, repositoryPath)
	if err != nil {
		logme.Debugln("Failed to call LLM:", err)
		return nil, err
	}

	logme.Debugln("LLM analysis completed successfully")
	return answers, nil
}

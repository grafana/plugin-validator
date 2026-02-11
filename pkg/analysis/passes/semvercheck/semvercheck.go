package semvercheck

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadatavalid"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
	"github.com/grafana/plugin-validator/pkg/llmclient"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/versioncommitfinder"
	"github.com/hashicorp/go-version"
)

//go:embed prompt.txt
var promptTemplate string

// SemVerAnalysisResponse represents the LLM response for SemVer analysis
type SemVerAnalysisResponse struct {
	HasBreakingChanges     bool     `json:"has_breaking_changes"`
	HasNewFeatures         bool     `json:"has_new_features"`
	HasBugFixes            bool     `json:"has_bug_fixes"`
	BreakingChangesList    []string `json:"breaking_changes_list"`
	NewFeaturesList        []string `json:"new_features_list"`
	BugFixesList           []string `json:"bug_fixes_list"`
	RecommendedVersionBump string   `json:"recommended_version_bump"`
	Explanation            string   `json:"explanation"`
}

var (
	semverMismatch = &analysis.Rule{
		Name:     "semver-mismatch",
		Severity: analysis.SuspectedProblem,
	}
	breakingChangeDetected = &analysis.Rule{
		Name:     "breaking-change-detected",
		Severity: analysis.SuspectedProblem,
	}
	semverAnalysisSkipped = &analysis.Rule{
		Name:     "semver-analysis-skipped",
		Severity: analysis.SuspectedProblem,
	}
)

// blockingAnalyzers contains validators that, if they report errors, should cause
// the SemVer analysis to be skipped
var blockingAnalyzers = []*analysis.Analyzer{
	archive.Analyzer,
	metadata.Analyzer,
	metadatavalid.Analyzer,
	modulejs.Analyzer,
}

var Analyzer = &analysis.Analyzer{
	Name:     "semvercheck",
	Requires: blockingAnalyzers,
	Run:      run,
	Rules:    []*analysis.Rule{semverMismatch, breakingChangeDetected, semverAnalysisSkipped},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "SemVer Compliance",
		Description: "Uses LLM to detect breaking changes and verify version increments match SemVer conventions.",
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
	// Check if any blocking analyzers reported errors
	for _, analyzer := range blockingAnalyzers {
		if pass.AnalyzerHasErrors(analyzer) {
			pass.ReportResult(
				pass.AnalyzerName,
				semverAnalysisSkipped,
				fmt.Sprintf("SemVer analysis skipped due to errors in %s", analyzer.Name),
				fmt.Sprintf(
					"Fix the errors reported by %s before SemVer analysis can run.",
					analyzer.Name,
				),
			)
			return nil, nil
		}
	}

	if pass.CheckParams.SourceCodeReference == "" {
		return nil, nil
	}

	if os.Getenv("SKIP_LLM_SEMVER") != "" {
		return nil, nil
	}

	if err := llmClient.CanUseLLM(); err != nil {
		return nil, nil
	}

	// Only support GitHub URLs
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

	// Need both versions to compare
	if versions.CurrentGrafanaVersion == nil || versions.SubmittedGitHubVersion == nil ||
		versions.CurrentGrafanaVersion.CommitSHA == "" || versions.SubmittedGitHubVersion.CommitSHA == "" {
		logme.Debugln("Cannot run SemVer analysis - missing version information")
		return nil, nil
	}

	// Parse versions
	currentVersion, err := version.NewVersion(versions.CurrentGrafanaVersion.Version)
	if err != nil {
		logme.Debugln("Failed to parse current version:", err)
		return nil, nil
	}

	newVersion, err := version.NewVersion(versions.SubmittedGitHubVersion.Version)
	if err != nil {
		logme.Debugln("Failed to parse new version:", err)
		return nil, nil
	}

	// Determine version bump type
	versionBumpType := determineVersionBumpType(currentVersion, newVersion)

	// Run LLM analysis to detect changes
	response, err := runSemVerLLMAnalysis(
		versions.SubmittedGitHubVersion.Version,
		versions.SubmittedGitHubVersion.CommitSHA,
		versions.CurrentGrafanaVersion.Version,
		versions.CurrentGrafanaVersion.CommitSHA,
		versions.RepositoryPath,
	)
	if err != nil {
		logme.Debugln("Failed to run SemVer LLM analysis:", err)
		return nil, nil
	}

	// Report breaking changes
	if response.HasBreakingChanges {
		breakingChangesDetail := formatBreakingChanges(response.BreakingChangesList)
		pass.ReportResult(
			pass.AnalyzerName,
			breakingChangeDetected,
			fmt.Sprintf("Breaking changes detected in version %s → %s",
				versions.CurrentGrafanaVersion.Version,
				versions.SubmittedGitHubVersion.Version),
			breakingChangesDetail,
		)
	}

	// Check for SemVer mismatch
	if response.HasBreakingChanges && versionBumpType != "major" {
		pass.ReportResult(
			pass.AnalyzerName,
			semverMismatch,
			"SemVer mismatch: Breaking changes require major version bump",
			fmt.Sprintf(
				"Breaking changes were detected, but the version was only bumped from %s to %s (%s bump). "+
					"According to SemVer, breaking changes require a major version bump (e.g., 1.x.x → 2.0.0).\n\n"+
					"**Detected breaking changes:**\n%s\n\n"+
					"**Recommendation:** %s",
				versions.CurrentGrafanaVersion.Version,
				versions.SubmittedGitHubVersion.Version,
				versionBumpType,
				formatBreakingChanges(response.BreakingChangesList),
				response.Explanation,
			),
		)
	} else if response.HasNewFeatures && !response.HasBreakingChanges && versionBumpType == "patch" {
		pass.ReportResult(
			pass.AnalyzerName,
			semverMismatch,
			"SemVer mismatch: New features typically require minor version bump",
			fmt.Sprintf(
				"New features were detected, but the version was only bumped from %s to %s (patch bump). "+
					"According to SemVer, new features typically require a minor version bump (e.g., 1.0.x → 1.1.0).\n\n"+
					"**Detected new features:**\n%s\n\n"+
					"**Recommendation:** %s",
				versions.CurrentGrafanaVersion.Version,
				versions.SubmittedGitHubVersion.Version,
				formatFeaturesList(response.NewFeaturesList),
				response.Explanation,
			),
		)
	}

	return response, nil
}

func determineVersionBumpType(current, new *version.Version) string {
	currentSegments := current.Segments()
	newSegments := new.Segments()

	// Ensure we have at least 3 segments
	for len(currentSegments) < 3 {
		currentSegments = append(currentSegments, 0)
	}
	for len(newSegments) < 3 {
		newSegments = append(newSegments, 0)
	}

	if newSegments[0] > currentSegments[0] {
		return "major"
	}
	if newSegments[1] > currentSegments[1] {
		return "minor"
	}
	if newSegments[2] > currentSegments[2] {
		return "patch"
	}

	return "none"
}

func formatBreakingChanges(changes []string) string {
	if len(changes) == 0 {
		return "No specific breaking changes identified."
	}
	var formatted []string
	for _, change := range changes {
		formatted = append(formatted, "- "+change)
	}
	return strings.Join(formatted, "\n")
}

func formatFeaturesList(features []string) string {
	if len(features) == 0 {
		return "No specific new features identified."
	}
	var formatted []string
	for _, feature := range features {
		formatted = append(formatted, "- "+feature)
	}
	return strings.Join(formatted, "\n")
}

func generatePrompt(newVersion, newCommit, currentVersion, currentCommit string) (string, error) {
	if newVersion == "" || newCommit == "" || currentVersion == "" || currentCommit == "" {
		return "", fmt.Errorf("version information incomplete")
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

func runSemVerLLMAnalysis(
	newVersion, newCommit, currentVersion, currentCommit, repositoryPath string,
) (*SemVerAnalysisResponse, error) {
	prompt, err := generatePrompt(newVersion, newCommit, currentVersion, currentCommit)
	if err != nil {
		logme.Debugln("Failed to generate prompt:", err)
		return nil, err
	}

	llmclient.CleanUpPromptFiles(repositoryPath)

	if err := llmClient.CallLLM(prompt, repositoryPath, nil); err != nil {
		logme.Debugln("Failed to call LLM:", err)
		return nil, err
	}

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

	var response SemVerAnalysisResponse
	if err := json.Unmarshal(responsesData, &response); err != nil {
		logme.Debugln("Failed to parse replies.json:", err)
		return nil, fmt.Errorf("failed to parse replies.json: %w", err)
	}

	logme.Debugln("SemVer LLM analysis completed successfully")
	return &response, nil
}

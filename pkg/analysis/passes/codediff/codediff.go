package codediff

import (
	"fmt"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/versioncommitfinder"
)

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
	Requires: []*analysis.Analyzer{readme.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{codeDiffAnalysis, codeDiffversions},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "Code Diff",
		Description:  "",
		Dependencies: "Google API Key with Generative AI access",
	},
}

func isGitHubURL(url string) bool {
	return strings.Contains(strings.ToLower(url), "github.com")
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.CheckParams.SourceCodeReference == "" {
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

	versions, err := versioncommitfinder.FindPluginVersionsRefs(
		pass.CheckParams.SourceCodeReference,
		"",
	)
	if err != nil {
		logme.Debugln("Failed to find versions", err)
		return nil, nil
	}

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

package versioncommitfinder

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/grafana"
	"github.com/grafana/plugin-validator/pkg/llmclient"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/repotool"
	"github.com/grafana/plugin-validator/pkg/utils"
	"github.com/tailscale/hujson"
)

type VersionComparison struct {
	PluginID               string                `json:"pluginId"`
	CurrentGrafanaVersion  *repotool.VersionInfo `json:"currentGrafanaVersion"`
	SubmittedGitHubVersion *repotool.VersionInfo `json:"submittedGitHubVersion"`
	Repository             *repotool.RepoInfo    `json:"repository"`
	RepositoryPath         string                `json:"repositoryPath"`
}

type PackageJson struct {
	Version string `json:"version"`
}

type geminiOutput struct {
	CommitSHA string `json:"commitSHA"`
	Reasoning string `json:"reasoning"`
}

// resolveVersion resolves the %VERSION% placeholder by reading package.json
func resolveVersion(archivePath, pluginVersion string) (string, error) {
	if pluginVersion != "%VERSION%" {
		return pluginVersion, nil
	}

	packageJsonPath := filepath.Join(archivePath, "package.json")
	rawPackageJson, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read package.json: %w", err)
	}

	// using hujson to allow tolerance in package.json (comments, trailing commas)
	stdPackageJson, err := hujson.Standardize(rawPackageJson)
	if err != nil {
		return "", fmt.Errorf("failed to standardize package.json: %w", err)
	}

	var packageJson PackageJson
	if err := json.Unmarshal(stdPackageJson, &packageJson); err != nil {
		return "", fmt.Errorf("failed to parse package.json: %w", err)
	}

	if packageJson.Version == "" {
		return "", fmt.Errorf("version not found in package.json")
	}

	return packageJson.Version, nil
}

func FindPluginVersionsRefs(
	githubURL string,
	// repoPath is often empty but you can pass it to speed up testing
	repoPath string,
) (*VersionComparison, func(), error) {
	logme.DebugFln(
		"Starting version comparison for GitHub URL: %s",
		githubURL,
	)

	archivePath := repoPath
	var cleanup func()

	if archivePath == "" {
		clonedPath, cleanupFn, err := repotool.CloneToTempWithDepth(githubURL, 100)
		if err != nil {
			logme.DebugFln("Failed to clone repo: %v", err)
			return nil, nil, fmt.Errorf("failed to clone repo: %w", err)
		}
		archivePath = clonedPath
		cleanup = cleanupFn
	}

	repoInfo, err := repotool.ParseRepoFromGitURL(githubURL)
	if err != nil {
		logme.DebugFln("Failed to parse GitHub URL: %v", err)
		return nil, nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}
	logme.DebugFln(
		"Parsed repo: owner: %s repo: %s (branch/tag: %s)",
		repoInfo.Owner,
		repoInfo.Repo,
		repoInfo.Ref,
	)

	logme.DebugFln("Fetching repository tags")
	// Fetch all tags so we can checkout to any tag ref (shallow clones don't include all tags)
	fetchTagsCmd := exec.Command("git", "fetch", "--tags")
	fetchTagsCmd.Dir = archivePath
	if err := fetchTagsCmd.Run(); err != nil {
		logme.DebugFln("Failed to fetch tags: %v", err)
	}

	// Try to fetch the specific ref in case it's a branch (shallow clones only have default branch)
	if repoInfo.Ref != "" {
		logme.DebugFln("Fetching ref: %s", repoInfo.Ref)
		fetchRefCmd := exec.Command("git", "fetch", "origin", repoInfo.Ref)
		fetchRefCmd.Dir = archivePath
		if err := fetchRefCmd.Run(); err != nil {
			logme.DebugFln("Failed to fetch ref %s (may not be a branch): %v", repoInfo.Ref, err)
		}
	}

	logme.DebugFln("Checking out ref: %s", repoInfo.Ref)
	if repoInfo.Ref != "" {
		cmd := exec.Command("git", "checkout", repoInfo.Ref)
		cmd.Dir = archivePath
		if err := cmd.Run(); err != nil {
			logme.DebugFln("Failed to checkout to ref %s: %v", repoInfo.Ref, err)
			return nil, nil, fmt.Errorf("failed to checkout to ref %s: %w", repoInfo.Ref, err)
		}
	} else {
		// make sure to checkout to main or master
		cmd := exec.Command("git", "checkout", "main")
		cmd.Dir = archivePath
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("git", "checkout", "master")
			cmd.Dir = archivePath
			if err := cmd.Run(); err != nil {
				logme.DebugFln("Failed to checkout to main or master: %v", err)
				return nil, nil, fmt.Errorf("failed to checkout to main or master: %w", err)
			}
		}
	}

	pluginMetadata, err := utils.GetPluginMetadata(archivePath)
	if err != nil {
		logme.DebugFln("Failed to extract plugin metadata: %v", err)
		return nil, nil, fmt.Errorf("failed to extract plugin metadata from archive: %w", err)
	}
	pluginID := pluginMetadata.ID
	rawPluginVersion := pluginMetadata.Info.Version

	// Resolve %VERSION% placeholder if present
	pluginVersion, err := resolveVersion(archivePath, rawPluginVersion)
	if err != nil {
		logme.DebugFln("Failed to resolve plugin version: %v", err)
		return nil, nil, fmt.Errorf("failed to resolve plugin version: %w", err)
	}

	var currentGrafanaVersion *repotool.VersionInfo

	grafanaClient := grafana.NewClient()
	grafanaVersions, err := grafanaClient.FindPluginVersions(pluginID)

	if err == nil && len(grafanaVersions) >= 1 {
		logme.DebugFln("Found %d Grafana API versions", len(grafanaVersions))
		grafanaAPIVersion := grafanaVersions[0]
		logme.DebugFln("Found Grafana API version: %s", grafanaAPIVersion.Version)

		currentGrafanaVersion, err = repotool.FindReleaseByVersion(
			repoInfo,
			grafanaAPIVersion.Version,
		)
		if err != nil {
			logme.DebugFln(
				"Could not find Grafana version %s in GitHub: %v",
				grafanaAPIVersion.Version,
				err,
			)
			currentGrafanaVersion = &repotool.VersionInfo{
				Version:   grafanaAPIVersion.Version,
				CommitSHA: grafanaAPIVersion.Commit,
				Source:    "grafana",
				CreatedAt: grafanaAPIVersion.CreatedAt,
				URL:       grafanaAPIVersion.URL,
			}
		}
	}
	logme.DebugFln("Current Grafana version: %s", currentGrafanaVersion)

	// If we still don't have a commit SHA, try gemini as a last resort
	if currentGrafanaVersion != nil && currentGrafanaVersion.CommitSHA == "" {
		logme.DebugFln("Attempting to find commit SHA using gemini CLI")
		if sha, err := findCommitWithGemini(archivePath, currentGrafanaVersion.Version); err == nil &&
			sha != "" {
			currentGrafanaVersion.CommitSHA = sha
			currentGrafanaVersion.Source = "gemini"
			currentGrafanaVersion.URL = fmt.Sprintf(
				"https://github.com/%s/%s/commit/%s",
				repoInfo.Owner,
				repoInfo.Repo,
				sha,
			)
			logme.DebugFln("Found commit SHA via gemini: %s", sha)
		} else if err != nil {
			logme.DebugFln("Gemini fallback failed: %v", err)
		}
	}

	// Get current commit SHA for the submitted version
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = archivePath
	commitOutput, err := cmd.Output()
	if err != nil {
		logme.DebugFln("Failed to get current commit SHA: %v", err)
		return nil, nil, fmt.Errorf("failed to get current commit SHA: %w", err)
	}
	currentCommitSHA := strings.TrimSpace(string(commitOutput))
	logme.DebugFln(
		"Current commit SHA for submitted version %s: %s",
		pluginVersion,
		currentCommitSHA,
	)

	submittedGitHubVersion := &repotool.VersionInfo{
		Version:   pluginVersion,
		CommitSHA: currentCommitSHA,
		URL: fmt.Sprintf(
			"https://github.com/%s/%s/commit/%s",
			repoInfo.Owner,
			repoInfo.Repo,
			currentCommitSHA,
		),
		Source: "current-archive",
	}

	return &VersionComparison{
		PluginID:               pluginID,
		CurrentGrafanaVersion:  currentGrafanaVersion,
		SubmittedGitHubVersion: submittedGitHubVersion,
		Repository:             repoInfo,
		RepositoryPath:         archivePath,
	}, cleanup, nil
}

// findCommitWithGemini uses gemini CLI as a last resort to find the commit SHA
// that corresponds to a specific version by analyzing git history.
func findCommitWithGemini(archivePath, version string) (string, error) {
	client := llmclient.NewGeminiClient()
	if err := client.CanUseLLM(); err != nil {
		return "", fmt.Errorf("cannot use LLM: %w", err)
	}

	logme.DebugFln("Using gemini CLI to find commit for version %s", version)

	// Build the prompt
	prompt := fmt.Sprintf(
		`You are in a git repository. Find the commit SHA for when version "%s was released".

IMPORTANT: Follow this EXACT process step by step. Explain what you are doing and what you find at each step.

## STEP 1: INVESTIGATION (find candidates)
Search for the version using multiple approaches:
- Check git tags: git tag -l "*%s*" or git tag -l
- Search commit messages: git log --oneline --grep="%s"
- Look for release patterns: git log --oneline --grep="release"
- Check recent history: git log --oneline -30

Different developers have different workflows, you must keep in consideration:
- Some bump the version first, then work on features, then package/release later
- Some implement changes first, then bump version at the end as the release commit
- CHANGELOGS are unreliable, always validate your findings

We want the commit that "truly" signifies when the version was "completed"

## STEP 2: VERIFICATION (mandatory - do not skip!)
Once you find a candidate commit, you MUST verify it by checking the actual SOURCE files:

git show <commit>:src/plugin.json | grep version
git show <commit>:package.json | grep version

IMPORTANT: IGNORE the dist/ folder - it contains build artifacts, not source files.

Check src/plugin.json as the SOURCE OF TRUTH for the version.
- If src/plugin.json has an actual version number, that version MUST match "%s"
- If src/plugin.json has a "%%VERSION%%" placeholder, THEN check package.json for the actual version
- package.json is sometimes bumped BEFORE src/plugin.json - if they differ, keep looking for when src/plugin.json was updated to "%s"

## STEP 3: DOUBLE-CHECK (mandatory - do not skip!)
Even if you found a matching commit, investigate AT LEAST ONE more approach:
- If you found it via tag, also check commit messages
- If you found it via commit message, also check tags
- Look at the commit BEFORE your candidate to see if version was already there
- Is this the commit that had a "full" version? Developers can bump version and keep pilling up commits after for the same version.

This helps catch cases where version was bumped earlier than the "release" commit.

## STEP 4: OUTPUT
Only after completing steps 1-3, write to output.json:

{
  "commitSHA": "full 40-character SHA (use 'git rev-parse <short-sha>' to expand if needed), or empty string if not found",
  "reasoning": "brief explanation including: how you found it, what verification you did, what double-check you performed? Is it the "complete" version commit"
}

IMPORTANT: The commitSHA MUST be the full 40-character hash, not a short hash.

YOU MUST validate your JSON by running: jq . output.json
If it fails, fix the JSON and try again.

Once output.json is valid, you are DONE. Exit immediately.`,

		version,
		version,
		version,
		version,
		version,
	)

	llmclient.CleanUpPromptFiles(archivePath)

	if err := client.CallLLM(prompt, archivePath, &llmclient.CallLLMOptions{
		// we are using gemini 2.5 flash  after testing it against gemini 3 flash
		// and see much better performance in speed and obeying instructions for this
		// particular case
		Model: "gemini-2.5-flash",
	}); err != nil {
		os.Remove(filepath.Join(archivePath, "output.json"))
		return "", fmt.Errorf("gemini CLI failed: %w", err)
	}

	// Read output.json
	outputPath := filepath.Join(archivePath, "output.json")
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		os.Remove(outputPath)
		return "", fmt.Errorf("failed to read gemini output: %w", err)
	}

	// Log the raw output for debugging
	logme.DebugFln("Gemini output.json content: %s", string(outputData))

	// Parse the output
	var output geminiOutput
	if err := json.Unmarshal(outputData, &output); err != nil {
		os.Remove(outputPath)
		return "", fmt.Errorf("failed to parse gemini output: %w", err)
	}

	logme.DebugFln("Gemini found commit %s: %s", output.CommitSHA, output.Reasoning)

	// Cleanup
	os.Remove(outputPath)

	// If we got a short SHA, try to expand it
	commitSHA := output.CommitSHA
	if len(commitSHA) > 0 && len(commitSHA) < 40 {
		logme.DebugFln("Got short SHA %s, expanding with git rev-parse", commitSHA)
		expandCmd := exec.Command("git", "rev-parse", commitSHA)
		expandCmd.Dir = archivePath
		if expandedOutput, err := expandCmd.Output(); err == nil {
			commitSHA = strings.TrimSpace(string(expandedOutput))
			logme.DebugFln("Expanded to full SHA: %s", commitSHA)
		}
	}

	// Validate the commit SHA (should be 40 hex characters)
	if len(commitSHA) != 40 {
		return "", fmt.Errorf("invalid commit SHA length: %d", len(commitSHA))
	}

	return commitSHA, nil
}

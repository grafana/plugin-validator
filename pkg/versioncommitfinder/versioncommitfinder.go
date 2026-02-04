package versioncommitfinder

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/grafana"
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

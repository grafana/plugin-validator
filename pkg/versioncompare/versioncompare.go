package versioncompare

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/grafana/plugin-validator/pkg/grafana"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/utils"
)

// VersionComparer handles version comparison between Grafana.com and GitHub
type VersionComparer struct {
	grafanaClient *grafana.Client
}

// New creates a new VersionComparer instance
func New() *VersionComparer {
	return &VersionComparer{
		grafanaClient: grafana.NewClient(),
	}
}

// CompareVersions compares versions between GitHub repository and Grafana.com
// githubURL: GitHub repository URL (e.g., https://github.com/owner/repo or https://github.com/owner/repo/tree/branch)
// archivePath: Local path to extracted plugin archive
func (vc *VersionComparer) CompareVersions(
	githubURL, archivePath string,
) (*VersionComparison, error) {
	logme.DebugFln(
		"Starting version comparison for GitHub URL: %s, Archive: %s",
		githubURL,
		archivePath,
	)

	pluginMetadata, err := utils.GetPluginMetadata(archivePath)
	if err != nil {
		logme.DebugFln("Failed to extract plugin metadata: %v", err)
		return nil, fmt.Errorf("failed to extract plugin metadata from archive: %w", err)
	}
	pluginID := pluginMetadata.ID
	pluginVersion := pluginMetadata.Info.Version
	logme.DebugFln("Found plugin ID: %s, version: %s", pluginID, pluginVersion)

	repoInfo, err := parseRepoFromGitURL(githubURL)
	if err != nil {
		logme.DebugFln("Failed to parse GitHub URL: %v", err)
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}
	logme.DebugFln(
		"Parsed repo: owner: %s repo: %s (branch/tag: %s)",
		repoInfo.Owner,
		repoInfo.Repo,
		repoInfo.Branch,
	)

	if repoInfo.Branch != "" {
		logme.DebugFln("Checking out to ref: %s", repoInfo.Branch)
		cmd := exec.Command("git", "checkout", repoInfo.Branch)
		cmd.Dir = archivePath
		if err := cmd.Run(); err != nil {
			logme.DebugFln("Failed to checkout to ref %s: %v", repoInfo.Branch, err)
			return nil, fmt.Errorf("failed to checkout to ref %s: %w", repoInfo.Branch, err)
		}
		logme.DebugFln("Successfully checked out to ref: %s", repoInfo.Branch)
	}

	var currentGrafanaVersion *VersionInfo = nil
	grafanaVersions, err := vc.grafanaClient.FindPluginVersions(pluginID)
	if err == nil && len(grafanaVersions) >= 1 {
		grafanaAPIVersion := grafanaVersions[0]
		logme.DebugFln("Found Grafana API version: %s", grafanaAPIVersion.Version)

		currentGrafanaVersion, err = findReleaseByVersion(
			repoInfo,
			grafanaAPIVersion.Version,
			archivePath,
		)
		if err != nil {
			logme.DebugFln(
				"Could not find Grafana version %s in GitHub: %v",
				grafanaAPIVersion.Version,
				err,
			)
			currentGrafanaVersion = FromGrafanaPluginVersion(grafanaAPIVersion)
		} else {
			logme.DebugFln("Found Grafana version %s in GitHub with commit: %s",
				currentGrafanaVersion.Version, currentGrafanaVersion.CommitSHA)
		}
	}
	if currentGrafanaVersion == nil {
		logme.Debugln("No current Grafana version found")
	}

	// Get current commit SHA for the submitted version
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = archivePath
	commitOutput, err := cmd.Output()
	if err != nil {
		logme.DebugFln("Failed to get current commit SHA: %v", err)
		return nil, fmt.Errorf("failed to get current commit SHA: %w", err)
	}
	currentCommitSHA := strings.TrimSpace(string(commitOutput))
	logme.DebugFln(
		"Current commit SHA for submitted version %s: %s",
		pluginVersion,
		currentCommitSHA,
	)

	submittedGitHubVersion := &VersionInfo{
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

	logme.Debugln("Version comparison completed successfully")

	return &VersionComparison{
		PluginID:               pluginID,
		CurrentGrafanaVersion:  currentGrafanaVersion,
		SubmittedGitHubVersion: submittedGitHubVersion,
		Repository:             repoInfo,
	}, nil
}

// FindVersionByTag finds a specific version in the GitHub repository by tag/release
func (vc *VersionComparer) FindVersionByTag(githubURL, tag string) (*VersionInfo, error) {
	repoInfo, err := parseRepoFromGitURL(githubURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}

	return findReleaseByVersion(repoInfo, tag, "")
}

// GetLatestGitHubVersion gets the latest release/tag from a GitHub repository
func (vc *VersionComparer) GetLatestGitHubVersion(githubURL string) (*VersionInfo, error) {
	repoInfo, err := parseRepoFromGitURL(githubURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}

	return getLatestGitHubVersion(repoInfo)
}

// GetCurrentGrafanaVersion gets the current published version from Grafana.com
func (vc *VersionComparer) GetCurrentGrafanaVersion(pluginID string) (*VersionInfo, error) {
	versions, err := vc.grafanaClient.FindPluginVersions(pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Grafana plugin versions: %w", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for plugin %s on Grafana.com", pluginID)
	}

	// Return the latest (first) version
	return FromGrafanaPluginVersion(versions[0]), nil
}

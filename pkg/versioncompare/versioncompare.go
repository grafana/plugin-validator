package versioncompare

import (
	"fmt"

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

	var currentGrafanaVersion *VersionInfo = nil
	grafanaVersions, err := vc.grafanaClient.FindPluginVersions(pluginID)
	if err == nil && len(grafanaVersions) >= 1 {
		grafanaAPIVersion := grafanaVersions[0]
		logme.DebugFln("Found Grafana API version: %s", grafanaAPIVersion.Version)

		currentGrafanaVersion, err = findReleaseByVersion(repoInfo, grafanaAPIVersion.Version)
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

	logme.DebugFln("Finding GitHub version matching plugin version: %s", pluginVersion)
	matchingGitHubVersion, err := findReleaseByVersion(repoInfo, pluginVersion)
	if err != nil {
		logme.DebugFln("Failed to find matching GitHub version: %v", err)
		return nil, fmt.Errorf("failed to find GitHub version matching %s: %w", pluginVersion, err)
	}
	logme.DebugFln(
		"Found matching GitHub version: %s (source: %s, commit: %s)",
		matchingGitHubVersion.Version,
		matchingGitHubVersion.Source,
		matchingGitHubVersion.CommitSHA,
	)

	logme.Debugln("Version comparison completed successfully")

	return &VersionComparison{
		PluginID:               pluginID,
		CurrentGrafanaVersion:  currentGrafanaVersion,
		SubmittedGitHubVersion: matchingGitHubVersion,
		Repository:             repoInfo,
	}, nil
}

// FindVersionByTag finds a specific version in the GitHub repository by tag/release
func (vc *VersionComparer) FindVersionByTag(githubURL, tag string) (*VersionInfo, error) {
	repoInfo, err := parseRepoFromGitURL(githubURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}

	return findReleaseByVersion(repoInfo, tag)
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

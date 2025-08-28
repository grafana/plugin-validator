package versioncommitfinder

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/grafana/plugin-validator/pkg/grafana"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/repotool"
	"github.com/grafana/plugin-validator/pkg/utils"
)

// VersionComparison represents the result of comparing versions between Grafana.com and GitHub
type VersionComparison struct {
	PluginID               string                `json:"pluginId"`
	CurrentGrafanaVersion  *repotool.VersionInfo `json:"currentGrafanaVersion"`
	SubmittedGitHubVersion *repotool.VersionInfo `json:"submittedGitHubVersion"`
	Repository             *repotool.RepoInfo    `json:"repository"`
}

// VersionComparer handles version comparison between Grafana.com and GitHub
type VersionComparer struct {
	grafanaClient *grafana.Client
}

func FindPluginVersionsRefs(
	githubURL string,
	// repoPath is often empty but you can pass it to speed up testing
	repoPath string,
) (*VersionComparison, error) {
	logme.DebugFln(
		"Starting version comparison for GitHub URL: %s",
		githubURL,
	)

	archivePath := repoPath

	if archivePath == "" {
		clonedPath, cleanup, err := repotool.CloneToTempWithDepth(githubURL, 0)
		if err != nil {
			logme.DebugFln("Failed to clone repo: %v", err)
			return nil, fmt.Errorf("failed to clone repo: %w", err)
		}
		archivePath = clonedPath
		defer cleanup()
	}

	pluginMetadata, err := utils.GetPluginMetadata(archivePath)
	if err != nil {
		logme.DebugFln("Failed to extract plugin metadata: %v", err)
		return nil, fmt.Errorf("failed to extract plugin metadata from archive: %w", err)
	}
	pluginID := pluginMetadata.ID
	pluginVersion := pluginMetadata.Info.Version
	logme.DebugFln("Found plugin ID: %s, version: %s", pluginID, pluginVersion)

	repoInfo, err := repotool.ParseRepoFromGitURL(githubURL)
	if err != nil {
		logme.DebugFln("Failed to parse GitHub URL: %v", err)
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}
	logme.DebugFln(
		"Parsed repo: owner: %s repo: %s (branch/tag: %s)",
		repoInfo.Owner,
		repoInfo.Repo,
		repoInfo.Ref,
	)

	if repoInfo.Ref != "" {
		logme.DebugFln("Checking out to ref: %s", repoInfo.Ref)
		cmd := exec.Command("git", "checkout", repoInfo.Ref)
		cmd.Dir = archivePath
		if err := cmd.Run(); err != nil {
			logme.DebugFln("Failed to checkout to ref %s: %v", repoInfo.Ref, err)
			return nil, fmt.Errorf("failed to checkout to ref %s: %w", repoInfo.Ref, err)
		}
		logme.DebugFln("Successfully checked out to ref: %s", repoInfo.Ref)
	} else {
		// make sure to checkout to main or master
		cmd := exec.Command("git", "checkout", "main")
		cmd.Dir = archivePath
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("git", "checkout", "master")
			cmd.Dir = archivePath
			if err := cmd.Run(); err != nil {
				logme.DebugFln("Failed to checkout to main or master: %v", err)
				return nil, fmt.Errorf("failed to checkout to main or master: %w", err)
			}
		}
		logme.DebugFln("Successfully checked out to main or master")
	}

	var currentGrafanaVersion *repotool.VersionInfo = nil

	grafanaClient := grafana.NewClient()
	grafanaVersions, err := grafanaClient.FindPluginVersions(pluginID)

	if err == nil && len(grafanaVersions) >= 1 {
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

	logme.Debugln("Version comparison completed successfully")

	return &VersionComparison{
		PluginID:               pluginID,
		CurrentGrafanaVersion:  currentGrafanaVersion,
		SubmittedGitHubVersion: submittedGitHubVersion,
		Repository:             repoInfo,
	}, nil
}

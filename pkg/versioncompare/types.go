package versioncompare

import (
	"time"

	"github.com/grafana/plugin-validator/pkg/githubapi"
	"github.com/grafana/plugin-validator/pkg/grafana"
)

// VersionComparison represents the result of comparing versions between Grafana.com and GitHub
type VersionComparison struct {
	PluginID                string       `json:"pluginId"`
	CurrentGrafanaVersion   *VersionInfo `json:"currentGrafanaVersion"`
	SubmittedGitHubVersion  *VersionInfo `json:"submittedGitHubVersion"`
	Repository              *RepoInfo    `json:"repository"`
}

// VersionInfo contains information about a specific version
type VersionInfo struct {
	Version   string    `json:"version"`
	CommitSHA string    `json:"commitSha"`
	Source    string    `json:"source"` // "grafana", "github_release", "github_tag"
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url"`
}


// RepoInfo contains parsed GitHub repository information
type RepoInfo struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch,omitempty"` // Optional, from tree URLs
	Tag    string `json:"tag,omitempty"`    // Optional, from tree URLs
	URL    string `json:"url"`
}

// GitHubTag represents a GitHub tag
type GitHubTag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// FromGrafanaPluginVersion converts a Grafana API PluginVersion to our VersionInfo
func FromGrafanaPluginVersion(pv grafana.PluginVersion) *VersionInfo {
	return &VersionInfo{
		Version:   pv.Version,
		CommitSHA: pv.Commit,
		Source:    "grafana",
		CreatedAt: pv.CreatedAt,
		URL:       pv.URL,
	}
}

// FromGitHubRelease converts a GitHub API Release to our VersionInfo
func FromGitHubRelease(release githubapi.Release) *VersionInfo {
	createdAt, _ := time.Parse(time.RFC3339, release.CreatedAt)
	return &VersionInfo{
		Version:   release.TagName,
		CommitSHA: release.TargetCommitish,
		Source:    "github_release",
		CreatedAt: createdAt,
		URL:       release.HTMLURL,
	}
}

// FromGitHubTag converts a GitHub API Tag to our VersionInfo
func FromGitHubTag(tag GitHubTag, htmlURL string) *VersionInfo {
	return &VersionInfo{
		Version:   tag.Name,
		CommitSHA: tag.Commit.SHA,
		Source:    "github_tag",
		CreatedAt: time.Time{}, // Tags don't have creation time in the API
		URL:       htmlURL,
	}
}
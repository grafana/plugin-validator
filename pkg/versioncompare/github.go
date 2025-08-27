package versioncompare

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/grafana/plugin-validator/pkg/githubapi"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/repotool"
)

// fetchGitHubReleases fetches releases for a GitHub repository using the existing githubapi pattern
func fetchGitHubReleases(owner, repo string) ([]githubapi.Release, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Use GitHub token if available (following existing pattern)
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var releases []githubapi.Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

// fetchGitHubTags fetches tags for a GitHub repository as fallback when no releases exist
func fetchGitHubTags(owner, repo string) ([]GitHubTag, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", owner, repo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Use GitHub token if available
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tags []GitHubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	return tags, nil
}

// parseRepoFromGitURL extracts owner/repo from a GitHub URL using existing repotool
func parseRepoFromGitURL(githubURL string) (*RepoInfo, error) {
	if !repotool.IsSupportedGitUrl(githubURL) {
		return nil, fmt.Errorf("unsupported or invalid GitHub URL: %s", githubURL)
	}

	// Use internal parsing to extract parts
	gitUrl, err := parseGitUrlInternal(githubURL)
	if err != nil {
		return nil, err
	}

	// Extract owner/repo from BaseUrl
	// BaseUrl format: https://github.com/owner/repo
	parts := strings.Split(strings.TrimPrefix(gitUrl.BaseUrl, "https://github.com/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL format: %s", githubURL)
	}

	repo := &RepoInfo{
		Owner:  parts[0],
		Repo:   parts[1],
		Branch: gitUrl.Ref, // Could be branch or tag
		URL:    githubURL,
	}

	return repo, nil
}

// parseGitUrlInternal mimics repotool.parseGitUrl but returns the struct
type gitUrlInternal struct {
	BaseUrl string
	Ref     string
	RootDir string
}

func parseGitUrlInternal(url string) (gitUrlInternal, error) {
	// Simple parsing - in production would use the actual repotool regex
	if strings.Contains(url, "/tree/") {
		parts := strings.Split(url, "/tree/")
		if len(parts) == 2 {
			refParts := strings.Split(parts[1], "/")
			return gitUrlInternal{
				BaseUrl: parts[0],
				Ref:     refParts[0],
				RootDir: strings.Join(refParts[1:], "/"),
			}, nil
		}
	}

	return gitUrlInternal{
		BaseUrl: url,
		Ref:     "",
		RootDir: "",
	}, nil
}

// getLatestGitHubVersion gets the latest version from GitHub, trying releases first, then tags
func getLatestGitHubVersion(repo *RepoInfo) (*VersionInfo, error) {
	logme.DebugFln("Fetching latest version for %s/%s", repo.Owner, repo.Repo)
	
	// First try to get releases
	logme.Debugln("Trying to fetch GitHub releases")
	releases, err := fetchGitHubReleases(repo.Owner, repo.Repo)
	if err == nil && len(releases) > 0 {
		logme.DebugFln("Found %d releases", len(releases))
		// Filter out pre-releases and drafts, get the latest
		for _, release := range releases {
			if !release.Draft && !release.Prerelease {
				logme.DebugFln("Using release: %s (target: %s)", release.TagName, release.TargetCommitish)
				return FromGitHubRelease(release), nil
			}
		}
		logme.Debugln("All releases are drafts or pre-releases, falling back to tags")
	} else {
		logme.DebugFln("No releases found or error: %v", err)
	}

	// Fallback to tags if no releases or only pre-releases/drafts
	logme.Debugln("Trying to fetch GitHub tags")
	tags, err := fetchGitHubTags(repo.Owner, repo.Repo)
	if err != nil {
		logme.DebugFln("Failed to fetch tags: %v", err)
		return nil, fmt.Errorf("failed to fetch releases and tags: %w", err)
	}

	if len(tags) == 0 {
		logme.Debugln("No tags found in repository")
		return nil, fmt.Errorf("no releases or tags found in repository")
	}

	logme.DebugFln("Found %d tags, using latest: %s (commit: %s)", len(tags), tags[0].Name, tags[0].Commit.SHA)
	// Return the first (latest) tag
	tagURL := fmt.Sprintf("https://github.com/%s/%s/tree/%s", repo.Owner, repo.Repo, tags[0].Name)
	return FromGitHubTag(tags[0], tagURL), nil
}

// findReleaseByVersion finds a specific release/tag by version string
func findReleaseByVersion(repo *RepoInfo, version string) (*VersionInfo, error) {
	// Try to find in releases first
	releases, err := fetchGitHubReleases(repo.Owner, repo.Repo)
	if err == nil {
		for _, release := range releases {
			if strings.EqualFold(release.TagName, version) ||
				strings.EqualFold(release.TagName, "v"+version) ||
				strings.EqualFold(strings.TrimPrefix(release.TagName, "v"), version) {
				return FromGitHubRelease(release), nil
			}
		}
	}

	// Fallback to tags
	tags, err := fetchGitHubTags(repo.Owner, repo.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to find version %s in releases and tags: %w", version, err)
	}

	for _, tag := range tags {
		if strings.EqualFold(tag.Name, version) ||
			strings.EqualFold(tag.Name, "v"+version) ||
			strings.EqualFold(strings.TrimPrefix(tag.Name, "v"), version) {
			tagURL := fmt.Sprintf(
				"https://github.com/%s/%s/tree/%s",
				repo.Owner,
				repo.Repo,
				tag.Name,
			)
			return FromGitHubTag(tag, tagURL), nil
		}
	}

	return nil, fmt.Errorf("version %s not found in repository", version)
}


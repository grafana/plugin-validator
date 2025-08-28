package repotool

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/githubapi"
)

func fetchGitHubReleases(owner, repo string) ([]githubapi.Release, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

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

func fetchGitHubTags(owner, repo string) ([]GitHubTag, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", owner, repo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

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

func ParseRepoFromGitURL(githubURL string) (*RepoInfo, error) {
	if !IsSupportedGitUrl(githubURL) {
		return nil, fmt.Errorf("unsupported or invalid GitHub URL: %s", githubURL)
	}

	gitUrl, err := ParseGitUrl(githubURL)
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
		Owner: parts[0],
		Repo:  parts[1],
		Ref:   gitUrl.Ref, // Could be branch, tag or commit
		URL:   githubURL,
	}

	return repo, nil
}

func FindReleaseByVersion(
	repo *RepoInfo,
	version string,
) (*VersionInfo, error) {
	// Try to find in releases first
	releases, err := fetchGitHubReleases(repo.Owner, repo.Repo)
	if err == nil {
		for _, release := range releases {
			if strings.EqualFold(release.TagName, version) ||
				strings.EqualFold(release.TagName, "v"+version) ||
				strings.EqualFold(strings.TrimPrefix(release.TagName, "v"), version) {
				createdAt, _ := time.Parse(time.RFC3339, release.CreatedAt)
				return &VersionInfo{
					Version:   release.TagName,
					CommitSHA: release.TargetCommitish,
					Source:    "github_release",
					CreatedAt: createdAt,
					URL:       release.HTMLURL,
				}, nil
			}
		}
	}

	// Fallback to tags
	tags, err := fetchGitHubTags(repo.Owner, repo.Repo)
	if err == nil {
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
				return &VersionInfo{
					Version:   tag.Name,
					CommitSHA: tag.Commit.SHA,
					Source:    "github_tag",
					CreatedAt: time.Time{}, // Tags don't have creation time in the API
					URL:       tagURL,
				}, nil
			}
		}
	}
	return &VersionInfo{
		Version:   version,
		CommitSHA: "",
		URL:       "",
	}, nil
}

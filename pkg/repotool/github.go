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

func fetchTagCommitSHA(owner, repo, tagName string) (string, error) {
	// First get the tag reference
	refURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/git/refs/tags/%s",
		owner,
		repo,
		tagName,
	)

	req, err := http.NewRequest("GET", refURL, nil)
	if err != nil {
		return "", err
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tagRef struct {
		Object struct {
			SHA  string `json:"sha"`
			Type string `json:"type"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagRef); err != nil {
		return "", err
	}

	// If it's a tag object, we need to fetch the tag to get the commit SHA
	if tagRef.Object.Type == "tag" {
		tagURL := fmt.Sprintf(
			"https://api.github.com/repos/%s/%s/git/tags/%s",
			owner,
			repo,
			tagRef.Object.SHA,
		)

		req, err := http.NewRequest("GET", tagURL, nil)
		if err != nil {
			return "", err
		}

		if token != "" {
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var tag struct {
			Object struct {
				SHA string `json:"sha"`
			} `json:"object"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tag); err != nil {
			return "", err
		}

		return tag.Object.SHA, nil
	}

	// If it's a commit object, return the SHA directly
	return tagRef.Object.SHA, nil
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
			// Normalize both the release tag and the version for comparison
			releaseVersion := strings.TrimPrefix(release.TagName, "v")
			searchVersion := strings.TrimPrefix(version, "v")

			if strings.EqualFold(releaseVersion, searchVersion) {
				createdAt, _ := time.Parse(time.RFC3339, release.CreatedAt)

				// Get the actual commit SHA from the tag, not the target_commitish
				// which often contains the branch name (e.g., "main")
				commitSHA := release.TargetCommitish // fallback
				if actualCommitSHA, err := fetchTagCommitSHA(repo.Owner, repo.Repo, release.TagName); err == nil {
					commitSHA = actualCommitSHA
				}

				return &VersionInfo{
					Version:   release.TagName,
					CommitSHA: commitSHA,
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
			// Normalize both the tag name and the version for comparison
			tagVersion := strings.TrimPrefix(tag.Name, "v")
			searchVersion := strings.TrimPrefix(version, "v")

			if strings.EqualFold(tagVersion, searchVersion) {
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
	}, fmt.Errorf("release for version %s not found", version)
}

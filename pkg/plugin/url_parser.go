package plugin

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

// ErrInvalidPullRequest is returned whenever a URL or commit couldn't be
// determined from a diff.
var ErrInvalidPullRequest = errors.New("invalid pull request")

// parseRef parses a references to a GitHub repository from a URL.
func parseRef(rawurl string) (Ref, error) { //nolint:golint,unused
	if !strings.HasPrefix(rawurl, "https://github.com/") {
		ref := "master"

		fields := strings.Split(rawurl, "@")
		if len(fields) == 2 {
			ref = fields[1]
		}

		slug := strings.Split(fields[0], "/")
		if len(slug) != 2 {
			return Ref{}, errors.New("unsupported path format")
		}

		return Ref{
			Username: slug[0],
			Repo:     slug[1],
			Ref:      ref,
		}, nil
	}

	path := strings.TrimPrefix(rawurl, "https://github.com/")
	parts := strings.Split(path, "/")

	// Check whether URL references a pull request.
	if strings.HasPrefix(path, "grafana/grafana-plugin-repository/pull/") {
		url, commit, err := findPluginByPR(parts[3])
		if err != nil {
			return Ref{}, err
		}

		path = strings.TrimPrefix(url+"/tree/"+commit, "https://github.com/")
		parts = strings.Split(path, "/")

		return Ref{
			Username: parts[0],
			Repo:     parts[1],
			Ref:      parts[3],
		}, nil
	}

	var ref string
	if len(parts) < 4 {
		ref = "master"
	} else {
		ref = parts[3]
	}

	return Ref{
		Username: parts[0],
		Repo:     parts[1],
		Ref:      ref,
	}, nil
}

// findPluginByPR parses the diff for a pull request and extracts the URL and
// commit SHA.
func findPluginByPR(pr string) (string, string, error) { //nolint:golint,unused
	url := fmt.Sprintf("https://api.github.com/repos/grafana/grafana-plugin-repository/pulls/%s", pr)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	return versionFromDiff(b)
}

// versionFromDiff looks for a url and commit property in the diff and returns
// their values.
func versionFromDiff(b []byte) (string, string, error) { //nolint:golint,unused
	urlMatches := regexp.MustCompile(`\+\s+"url":\s"(.+)"`).FindAllSubmatch(b, -1)
	if len(urlMatches) < 1 {
		return "", "", ErrInvalidPullRequest
	}

	commitMatches := regexp.MustCompile(`\+\s+"commit":\s"(.+)"`).FindAllSubmatch(b, -1)
	if len(commitMatches) != 1 {
		return "", "", ErrInvalidPullRequest
	}

	return string(urlMatches[0][1]), string(commitMatches[0][1]), nil
}

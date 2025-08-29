package repotool

import (
	"time"
)

type VersionInfo struct {
	Version   string    `json:"version"`
	CommitSHA string    `json:"commitSha"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url"`
}

type RepoInfo struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Ref   string `json:"branch,omitempty"`
	Tag   string `json:"tag,omitempty"`
	URL   string `json:"url"`
}

type GitHubTag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

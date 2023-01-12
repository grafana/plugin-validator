package repotool

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

type GitUrl struct {
	BaseUrl string
	Ref     string
	RootDir string
}

func CloneToTempDir(uri string, ref string) (string, func(), error) {

	var err error

	err = checkDependencies()
	if err != nil {
		return "", nil, err
	}

	// create a tmp dir
	tmpDir, err := os.MkdirTemp("", "validator")
	if err != nil {
		return "", nil, err
	}

	// construct command to clone
	cmd := []string{"git", "clone", "--depth", "1", uri, tmpDir}
	if ref != "" {
		// git --branch takes cares of figuring if is a branch or a tag
		cmd = []string{"git", "clone", "--depth", "1", "--branch", ref, uri, tmpDir}
	}

	systemCommand := exec.Command(cmd[0], cmd[1:]...)
	systemCommand.Stdout = os.Stdout
	systemCommand.Stderr = os.Stderr

	err = systemCommand.Run()
	if err != nil {
		return "", nil, fmt.Errorf("couldn't clone repo: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	if err != nil {
		fmt.Println("error: ", err)
		cleanup()
		return "", nil, err
	}

	return tmpDir, cleanup, nil
}

func GitUrlToLocalPath(url string) (string, func(), error) {
	parsedGitUrl, err := parseGitUrl(url)
	if err != nil {
		return "", nil, err
	}

	tmpDir, cleanup, err := CloneToTempDir(parsedGitUrl.BaseUrl, parsedGitUrl.Ref)
	if err != nil {
		return "", cleanup, err
	}

	rootDir := fmt.Sprintf("%s/%s", tmpDir, parsedGitUrl.RootDir)
	fmt.Printf("rootDir: %s", rootDir)
	return rootDir, cleanup, nil

}

// regexes of supported git repositories
// group 1: git clone url
// group 3: ref (might be empty)
// group 4: root dir (might be empty)
var servicesRe []*regexp.Regexp = []*regexp.Regexp{
	// bitbucket
	regexp.MustCompile(`(?i)^(https:\/\/bitbucket\.org\/[^/]+\/[^/]+\/)(src\/([^/]*)\/?(.*)$)?`),
	// gitlab
	regexp.MustCompile(`(?i)^(https:\/\/gitlab\.com\/[^/]+\/[^/]+\/)(-\/tree\/([^/]*)\/?(.*)$)?`),
	// github
	regexp.MustCompile(`(?i)^(https:\/\/github\.com\/[^/]+\/[^/]+\/)(tree\/([^/]*)\/?(.*)$)?`),
}

func parseGitUrl(url string) (GitUrl, error) {
	var match []string

	for _, re := range servicesRe {
		match = re.FindStringSubmatch(url)
		if len(match) > 0 {
			break
		}
	}

	if len(match) > 0 {
		return GitUrl{
			BaseUrl: match[1],
			Ref:     match[3],
			RootDir: match[4],
		}, nil
	}

	return GitUrl{}, fmt.Errorf("couldn't parse git url: %s. This git service is not supported.", url)

}

func checkDependencies() error {
	// check that git command exists
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git command not found. You need to install git to use the source code flag")
	}
	return nil
}

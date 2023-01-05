package repotool

import (
	"os"

	"github.com/go-git/go-git/v5"
)

func CloneToTempDir(uri string) (string, func(), error) {
	// create a tmp dir
	tmpDir, err := os.MkdirTemp("", "validator")
	if err != nil {
		return "", nil, err
	}
	_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:      uri,
		Progress: os.Stdout,
		Depth:    1, // only clone the latest commit
	})

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	if err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpDir, cleanup, nil
}

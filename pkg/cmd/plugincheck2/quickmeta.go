package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// GetIDAndVersion Gets plugin id and version from extracted zip
func GetIDAndVersion(archiveDir string) (string, string, error) {
	// check extracted archive directory first
	dirInfo, err := os.Stat(archiveDir)
	// make sure the path exists
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, fmt.Errorf("path does not exist (%s): %w", archiveDir, err))
		return "", "", err
	}
	// make sure it is a directory
	if !dirInfo.IsDir() {
		fmt.Fprintln(os.Stderr, fmt.Errorf("path is not a directory (%s): %w", archiveDir, err))
		return "", "", err
	}
	fis, _ := ioutil.ReadDir(archiveDir)
	// check if there is a top-level directory
	if !fis[0].IsDir() {
		fmt.Fprintln(os.Stderr, fmt.Errorf("extracted zip does not have a subdirectory (%s)", archiveDir))
		return "", "", fmt.Errorf("extracted zip does not have a subdirectory (%s)", archiveDir)
	}
	// check file exists
	filename := filepath.Join(archiveDir, fis[0].Name(), "plugin.json")
	fileInfo, err := os.Stat(filename)
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, fmt.Errorf("plugin.json not found at %s: %w", filename, err))
		return "", "", err
	}
	// make sure it is a file
	mode := fileInfo.Mode()
	if !mode.IsRegular() {
		fmt.Fprintln(os.Stderr, fmt.Errorf("plugin.json is not a file: %s", filename))
		return "", "", err
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("cannot read plugin.json: %w", err))
		return "", "", err
	}
	var data struct {
		ID   string `json:"id"`
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("couldn't get plugin meta: %w", err))
		return "nil", "", err
	}
	return data.ID, data.Info.Version, nil
}

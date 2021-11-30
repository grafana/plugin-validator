package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// GetIDAndVersion Gets plugin id and version from extracted zip
func GetIDAndVersion(archiveDir string) (*string, *string, error) {
	fis, _ := ioutil.ReadDir(archiveDir)
	b, err := ioutil.ReadFile(filepath.Join(archiveDir, fis[0].Name(), "plugin.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("cannot read plugin.json: %w", err))
		return nil, nil, err
	}
	var data struct {
		ID   string `json:"id"`
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("couldn't get plugin meta: %w", err))
		return nil, nil, err
	}
	return &data.ID, &data.Info.Version, nil
}

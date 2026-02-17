package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

type BarebonePluginJson struct {
	Id string `json:"id"`
}

// GetPluginId returns the plugin id from the plugin.json file
// in the archive directory
//
// The plugin.json file might not be in the root directory
// at this point in the validator there's no certainty that the
// plugin.json file even exists
func GetPluginId(archiveDir string) (string, error) {
	if len(archiveDir) == 0 || archiveDir == "/" {
		return "", fmt.Errorf("archiveDir is empty")
	}
	pluginJsonPath, err := doublestar.FilepathGlob(archiveDir + "/**/plugin.json")
	if err != nil || len(pluginJsonPath) == 0 {
		return "", fmt.Errorf("Error getting plugin.json path: %s", err)
	}

	pluginJsonContent, err := os.ReadFile(pluginJsonPath[0])
	if err != nil {
		return "", err
	}

	// Unmarshal plugin.json
	var pluginJson BarebonePluginJson
	err = json.Unmarshal(pluginJsonContent, &pluginJson)
	if err != nil {
		return "", err
	}
	return pluginJson.Id, nil
}

// GetPluginMetadata returns the full plugin metadata from the plugin.json file
func GetPluginMetadata(archiveDir string) (*metadata.Metadata, error) {
	if len(archiveDir) == 0 || archiveDir == "/" {
		return nil, fmt.Errorf("archiveDir is empty")
	}
	pluginJsonPath, err := doublestar.FilepathGlob(archiveDir + "/**/plugin.json")
	if err != nil || len(pluginJsonPath) == 0 {
		return nil, fmt.Errorf("Error getting plugin.json path: %s", err)
	}

	pluginJsonContent, err := os.ReadFile(pluginJsonPath[0])
	if err != nil {
		return nil, err
	}

	// Unmarshal plugin.json
	var pluginJson metadata.Metadata
	err = json.Unmarshal(pluginJsonContent, &pluginJson)
	if err != nil {
		return nil, err
	}
	return &pluginJson, nil
}

// HasProperArchiveStructure checks if the archive has the proper structure:
// single top-level directory containing plugin.json
func HasProperArchiveStructure(archiveDir string) bool {
	fis, err := os.ReadDir(archiveDir)
	if err != nil || len(fis) == 0 {
		return false
	}

	// Check if first entry is a directory
	if !fis[0].IsDir() {
		return false
	}

	// Check if plugin.json exists in that directory
	pluginJsonPath := filepath.Join(archiveDir, fis[0].Name(), "plugin.json")
	if _, err := os.Stat(pluginJsonPath); err != nil {
		return false
	}

	return true
}

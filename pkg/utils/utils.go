package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bmatcuk/doublestar/v4"
)

type BarebonePluginJson struct {
	Id string `json:"id"`
}

/*
* GetPluginId returns the plugin id from the plugin.json file
* in the archive directory
*
* The plugin.json file might not be in the root directory
* at this point in the validator there's no certainty that the
* plugin.json file even exists
 */
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
	//unmarshal plugin.json
	var pluginJson BarebonePluginJson
	err = json.Unmarshal(pluginJsonContent, &pluginJson)
	if err != nil {
		return "", err
	}
	return pluginJson.Id, nil
}

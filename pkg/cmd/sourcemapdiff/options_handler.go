package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/archivetool"
	"github.com/grafana/plugin-validator/pkg/repotool"
)

func procesSourceCode(sourceCodeUri string) (string, func(), error) {
	sourceCodePath, sourceCodeCleanup, err := sourceCodeUriToLocalPath(sourceCodeUri)

	// check if there's a src/plugin.json file
	pluginJSONPath := filepath.Join(sourceCodePath, "src", "plugin.json")
	if _, err := os.Stat(pluginJSONPath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("src/plugin.json file not found in %s", pluginJSONPath)
	}

	return sourceCodePath, sourceCodeCleanup, err
}

func sourceCodeUriToLocalPath(sourceCodeUri string) (string, func(), error) {
	var (
		sourceCodePath string
		cleanup        func() = func() {}
		err            error
	)
	// if sourceCode is a zip, let archivetool handle it
	if filepath.Ext(sourceCodeUri) == ".zip" {
		sourceCodePath, cleanup, err = archivetool.ArchiveToLocalPath(sourceCodeUri)
		if err != nil {
			return "", nil, err
		}
		return sourceCodePath, cleanup, nil
	}

	// if it is an url to a git repo, clone it
	if (len(sourceCodeUri) > 4 && sourceCodeUri[:4] == "http") || (len(sourceCodeUri) > 3 && sourceCodeUri[:3] == "git") {
		fmt.Println(":: URL found. Trying to clone repository...")
		sourceCodePath, cleanup, err = repotool.CloneToTempDir(sourceCodeUri)
		if err != nil {
			return "", nil, err
		}
		return sourceCodePath, cleanup, nil
	}

	// if it is a local path, check if it exists
	if _, err := os.Stat(sourceCodeUri); os.IsNotExist(err) {
		return "", nil, err
	}
	sourceCodePath = sourceCodeUri
	return sourceCodePath, cleanup, nil
}

func getLocalPathFromArchiveOption(uri string) (string, func(), error) {
	var (
		archivePath string
		// init a noop cleanup
		archiveCleanup func() = func() {}
		err            error
	)

	// if archivePath ends with .zip, then it's a zip file let archivetool handle it
	if filepath.Ext(uri) == ".zip" {
		archivePath, archiveCleanup, err = archivetool.PluginArchiveToTempDir(uri)
		if err != nil {
			return "", nil, err
		}
		// else is a path to a directory
	} else {
		// check if exists in the filesystem
		if _, err := os.Stat(uri); os.IsNotExist(err) {
			return "", nil, err
		}
		archivePath = uri
	}

	// validate there's a plugin.json file
	pluginJSONPath := filepath.Join(archivePath, "plugin.json")
	if _, err := os.Stat(pluginJSONPath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("plugin.json file not found. A plugin must contain a plugin.json file at the root")
	}

	// check there's a .map file in archivePath
	files, err := os.ReadDir(archivePath)
	if err != nil {
		return "", nil, err
	}
	hasMapFile := false
	// var mapFile string
	for _, file := range files {
		if file.Name() == "module.js.map" {
			hasMapFile = true
			break
		}
	}

	if !hasMapFile {
		return "", nil, fmt.Errorf("module.js.map file not found in archive. You can't use this tool if the plugin doesn't have a source map file")
	}

	return archivePath, archiveCleanup, nil
}

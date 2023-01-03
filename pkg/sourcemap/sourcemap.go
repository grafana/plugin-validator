package sourcemap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

type rawSourceMap struct {
	Version        int      `json:"version"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
}

type sourceMap struct {
	Version      int
	Sources      map[string]string
	SourcesNames []string
}

var ignoreStartingWith = []string{
	"external ",
	"webpack/",
	"../node_modules",
	"./node_modules/",
}

var replaceRegex = regexp.MustCompile(`^webpack:\/*`)

func ParseSourceMapFromPath(sourceMapPath string) (*sourceMap, error) {
	sourceMapContent, err := os.ReadFile(sourceMapPath)
	if err != nil {
		return nil, err
	}
	return ParseSourceMapFromBytes(sourceMapContent)
}

func ParseSourceMapFromBytes(data []byte) (*sourceMap, error) {
	var parsedSourceMap rawSourceMap
	err := json.Unmarshal(data, &parsedSourceMap)
	if err != nil {
		return nil, err
	}

	var sourceMap sourceMap
	sourceMap.Version = parsedSourceMap.Version
	sourceMap.Sources = make(map[string]string)
	for i, sourceName := range parsedSourceMap.Sources {
		fileName := replaceRegex.ReplaceAllString(sourceName, "")
		if isIgnoredFile(fileName) {
			continue
		}
		sourceMap.Sources[fileName] = parsedSourceMap.SourcesContent[i]
	}
	return &sourceMap, nil
}

func ExtractSourceMapToPath(sourceMapPath string) (string, error) {
	sourceMapContent, err := os.ReadFile(sourceMapPath)
	if err != nil {
		return "", err
	}

	// parse source map
	sourceMapParsed, err := ParseSourceMapFromBytes(sourceMapContent)

	// create a temporal dir to extract the source map
	tmpSourceMapPath, err := os.MkdirTemp(os.TempDir(), "plugin-validator")

	for sourceName, sourceContent := range sourceMapParsed.Sources {

		// create the folder structure for the file
		fileParentFolder := filepath.Dir(filepath.Join(tmpSourceMapPath, sourceName))
		err = os.MkdirAll(fileParentFolder, 0755)
		if err != nil {
			return "", err
		}

		// write the file
		err := os.WriteFile(filepath.Join(tmpSourceMapPath, sourceName), []byte(sourceContent), 0644)
		if err != nil {
			return "", err
		}
	}

	return tmpSourceMapPath, nil
}

func isIgnoredFile(sourceName string) bool {
	// ignore external and webpack bootstrapf iles
	ignore := false
	for _, ignoreStart := range ignoreStartingWith {
		if len(sourceName) > len(ignoreStart) && sourceName[:len(ignoreStart)] == ignoreStart {
			return true
		}
	}
	return ignore
}

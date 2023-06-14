package sourcemap

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	var rawSourceMap rawSourceMap
	err := json.Unmarshal(data, &rawSourceMap)
	if err != nil {
		return nil, err
	}

	if len(rawSourceMap.Sources) == 0 {
		return nil, errors.New("generated source code map requires the original source file paths to be present in the sources property")
	}

	if len(rawSourceMap.SourcesContent) == 0 {
		return nil, errors.New("generated source code map requires the original source code to be present in the sourcesContent property")
	}

	if len(rawSourceMap.Sources) != len(rawSourceMap.SourcesContent) {
		return nil, errors.New("generated source code map requires the number of original source file paths (sources) to match with the number of original source code (sourcesContent)")
	}

	parseSourceMap := sourceMap{
		Version: rawSourceMap.Version,
		Sources: map[string]string{},
	}
	for i, sourceName := range rawSourceMap.Sources {
		fileName := replaceRegex.ReplaceAllString(sourceName, "")
		if isIgnoredFile(fileName) {
			continue
		}

		parseSourceMap.Sources[fileName] = rawSourceMap.SourcesContent[i]
	}
	return &parseSourceMap, nil
}

func ExtractSourceMapToPath(sourceMapPath string) (string, error) {
	// parse source map
	sourceMapParsed, err := ParseSourceMapFromPath(sourceMapPath)
	if err != nil {
		return "", err
	}

	// create a temporal dir to extract the source map
	tmpSourceMapPath, err := os.MkdirTemp(os.TempDir(), "plugin-validator")
	if err != nil {
		return "", err
	}

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
	// ignore css files
	// remove anything after a ? to ignore query params
	if strings.Contains(sourceName, "?") {
		sourceName = sourceName[:strings.Index(sourceName, "?")]
	}
	if strings.Contains(sourceName, "/node_modules/.pnpm/") {
		return true
	}
	var ignoredExtensions = []string{".css", ".scss", ".sass", ".less", ".svg", ".json"}
	for _, ext := range ignoredExtensions {
		if strings.HasSuffix(sourceName, ext) {
			return true
		}
	}
	// ignore external and webpack bootstrap iles
	ignore := false
	for _, ignoreStart := range ignoreStartingWith {
		if len(sourceName) > len(ignoreStart) && sourceName[:len(ignoreStart)] == ignoreStart {
			return true
		}
	}
	return ignore
}

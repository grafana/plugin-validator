package difftool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	godiffpatch "github.com/sourcegraph/go-diff-patch"

	"github.com/grafana/plugin-validator/pkg/sourcemap"
)

type DiffReport struct {
	SourceCodeMapPath string
	SourceCodePath    string
	Sources           map[string]*sourceDiff
	TotalDifferences  int
}

type sourceDiff struct {
	SourceCodePath           string
	SourceCodeMapPath        string
	SourceCodeFileContent    *string
	SourceCodeMapFileContent *string
	Diff                     *string
	FileFound                bool
	Equal                    bool
}

/*
CompareSourceMapToSourceCode compares the source code map to the source code.
It returns a DiffReport that contains the differences between the source code map and the source code.
sourceCodeMapFile is the path to the source code map file. (.js.map)
sourceCodePath is the path to the source code directory. (the directory that contains the source code files)
*/
func CompareSourceMapToSourceCode(
	pluginID string,
	sourceCodeMapFile string,
	sourceCodePath string,
) (DiffReport, error) {
	report := DiffReport{
		SourceCodeMapPath: sourceCodeMapFile,
		SourceCodePath:    sourceCodePath,
		Sources:           map[string]*sourceDiff{},
	}

	sourceCode, err := sourcemap.ParseSourceMapFromPath(pluginID, sourceCodeMapFile)
	if err != nil {
		return report, err
	}

	for sourceMapFileName, sourceMapContent := range sourceCode.Sources {
		sourceCodeFilePath := filepath.Join(sourceCodePath, sourceMapFileName)
		sourceMapContent := sourceMapContent

		sourceDiffReport := sourceDiff{
			SourceCodePath:           sourceCodeFilePath,
			SourceCodeMapPath:        sourceMapFileName,
			SourceCodeMapFileContent: &sourceMapContent,
			FileFound:                true,
			Equal:                    false,
		}

		report.Sources[sourceMapFileName] = &sourceDiffReport

		sourceCodeFileContent, err := os.ReadFile(sourceCodeFilePath)
		if err != nil {
			sourceDiffReport.FileFound = false
			report.TotalDifferences++
			continue
		}

		// Normalize line endings to Unix-style (LF) to ensure consistent comparison
		cleanSourceCodeFileContent := strings.ReplaceAll(
			string(sourceCodeFileContent),
			"\r\n",
			"\n",
		)
		sourceMapContent = strings.ReplaceAll(sourceMapContent, "\r\n", "\n")

		delta := godiffpatch.GeneratePatch(
			sourceMapFileName,
			cleanSourceCodeFileContent,
			sourceMapContent,
		)
		if len(delta) == 0 {
			sourceDiffReport.Equal = true
		} else {
			report.TotalDifferences++
		}
		sourceDiffReport.Diff = &delta
	}

	return report, nil
}

func (r *DiffReport) GeneratePrintableReport() string {
	var report string

	if r.TotalDifferences > 0 {
		report += fmt.Sprintf(
			"\nThe following %d file(s) differ when comparing source map with source code.\n",
			r.TotalDifferences,
		)
		for sourceMapFileName, source := range r.Sources {
			if source.Equal {
				continue
			}
			report += fmt.Sprintf(" - %s\n", sourceMapFileName)
		}
	} else {
		report += "No differences found in the comparable files\n"
		report += "\nBe aware that the source code map doesn't contain all of the files that are present in the source code. For example typescript types and node_modules dependencies.\n"
	}
	return report
}

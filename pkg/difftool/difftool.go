package difftool

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/sourcemap"
	godiffpatch "github.com/sourcegraph/go-diff-patch"
)

type diffReport struct {
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
It returns a diffReport that contains the differences between the source code map and the source code.
sourceCodeMapFile is the path to the source code map file. (.js.map)
sourceCodePath is the path to the source code directory. (the directory that contains the source code files)
*/
func CompareSourceMapToSourceCode(sourceCodeMapFile string, sourceCodePath string) (diffReport, error) {
	report := diffReport{
		SourceCodeMapPath: sourceCodeMapFile,
		SourceCodePath:    sourceCodePath,
		Sources:           map[string]*sourceDiff{},
	}

	sourceCode, err := sourcemap.ParseSourceMapFromPath(sourceCodeMapFile)
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

		delta := godiffpatch.GeneratePatch(sourceMapFileName, string(sourceCodeFileContent), sourceMapContent)
		if len(delta) == 0 {
			sourceDiffReport.Equal = true
		} else {
			report.TotalDifferences++
		}
		sourceDiffReport.Diff = &delta
	}

	return report, nil
}

func (r *diffReport) GeneratePrintableReport() string {
	var report string

	report += fmt.Sprintf("Source code map path: %s\n", r.SourceCodeMapPath)
	report += fmt.Sprintf("Source code path: %s\n", r.SourceCodePath)

	if r.TotalDifferences > 0 {
		report += fmt.Sprintf("\nFound %d file with differences\n", r.TotalDifferences)
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

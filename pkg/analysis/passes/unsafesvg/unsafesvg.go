package unsafesvg

import (
	"fmt"
	"os"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/svgvalidate"
)

var (
	unsafeSvgFile = &analysis.Rule{Name: "unsafe-svg-file", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "manifest",
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{unsafeSvgFile},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	svgValidator := svgvalidate.NewValidator()

	// find all svg
	svgFiles, err := doublestar.FilepathGlob(archiveDir + "/**/*.svg")
	if err != nil {
		return nil, err
	}

	for _, svgFile := range svgFiles {
		svgContent, err := os.ReadFile(svgFile)
		if err != nil {
			return nil, err
		}
		err = svgValidator.Validate(svgContent)
		if err != nil {
			pass.ReportResult(pass.AnalyzerName, unsafeSvgFile, fmt.Sprintf("SVG file %s is unsafe", svgFile), err.Error())
		}

	}

	return nil, nil
}

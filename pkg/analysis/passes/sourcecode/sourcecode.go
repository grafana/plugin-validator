package sourcecode

import (
	"fmt"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	sourceCodeNotProvided     = &analysis.Rule{Name: "source-code-not-provided", Severity: analysis.Warning}
	sourceCodeNotFound        = &analysis.Rule{Name: "source-code-not-found", Severity: analysis.Error}
	sourceCodeVersionMisMatch = &analysis.Rule{Name: "source-code-version-mismatch", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "sourcecode",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{sourceCodeNotFound, sourceCodeVersionMisMatch, sourceCodeNotProvided},
}

func run(pass *analysis.Pass) (interface{}, error) {

	sourceCodeDir := pass.SourceCodeDir
	if sourceCodeDir == "" {
		// If no source code dir is provided, only report the result if ReportAll is set, for backwards compatibility
		if sourceCodeNotProvided.ReportAll {
			pass.ReportResult(pass.AnalyzerName, sourceCodeNotProvided, fmt.Sprintf("Source code not provided or the provided URL %s does not point to a valid source code repository", pass.SourceCodeDir), "If you are passing a Git ref or sub-directory in the URL make sure they are correct.")
		}
		return nil, nil
	}

	return sourceCodeDir, nil
}

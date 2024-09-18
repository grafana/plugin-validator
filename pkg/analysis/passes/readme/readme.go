package readme

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
)

var (
	missingReadme = &analysis.Rule{Name: "missing-readme", Severity: analysis.Error}
	readmeComment = &analysis.Rule{Name: "readme-comment", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "readme",
	Run:      run,
	Requires: []*analysis.Analyzer{archive.Analyzer},
	Rules:    []*analysis.Rule{missingReadme},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	b, err := os.ReadFile(filepath.Join(archiveDir, "README.md"))
	if err != nil {
		if os.IsNotExist(err) {
			pass.ReportResult(pass.AnalyzerName, missingReadme, "missing README.md", "A README.md file is required for plugins. The contents of the file will be displayed in the Plugin catalog.")
			return nil, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		pass.ReportResult(pass.AnalyzerName, missingReadme, "README.md is empty", "A README.md file is required for plugins. The contents of the file will be displayed in the Plugin catalog.")
		return nil, nil
	}
	if missingReadme.ReportAll {
		missingReadme.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, missingReadme, "README.md: exists", "")
	}

	readmeContent := string(b)

	commentRegex := `<!--(.*?)-->`
	re := regexp.MustCompile(commentRegex)

	// No need find all
	comment := re.FindString(readmeContent)

	if len(comment) > 0 {
		pass.ReportResult(pass.AnalyzerName, readmeComment, "README.md contains comment(s).", "")
	}
	if readmeComment.ReportAll {
		readmeComment.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, readmeComment, "README.md does not contain any comments.", "")
	}

	return b, nil
}

package analysis

import (
	"fmt"

	"github.com/grafana/plugin-validator/pkg/logme"
)

type Severity string

var (
	Error            Severity = "error"
	Warning          Severity = "warning"
	OK               Severity = "ok"
	SuspectedProblem Severity = "suspected"
	Recommendation   Severity = "recommendation"
)

type Pass struct {
	AnalyzerName     string
	RootDir          string
	CheckParams      CheckParams
	ResultOf         map[*Analyzer]any
	Report           func(string, Diagnostic)
	Diagnostics      *Diagnostics
	AnalyzerRulesMap map[string]string
}

type CheckParams struct {
	// ArchiveFile contains the path passed to the validator. can be a file or a url
	ArchiveFile string
	// ArchiveDir contains the path to the extracted files from the ArchiveFile
	ArchiveDir string
	// SourceCodeDir contains the path to the plugin source code
	SourceCodeDir string
	// SourceCodeReference contains the reference passed to the validator as source code, can be a folder or an url
	SourceCodeReference string
	// Checksum contains the checksum passed to the validator as an argument
	Checksum string
	// ArchiveCalculatedMD5 contains the md5 checksum calculated from the archive
	ArchiveCalculatedMD5 string
	// ArchiveCalculatedSHA1 contains the sha1 checksum calculated from the archive
	ArchiveCalculatedSHA1 string
}

func (p *Pass) ReportResult(analysisName string, rule *Rule, message string, detail string) {
	if rule.Disabled {
		logme.Debugln(fmt.Sprintf("Rule %s is disabled. Skipping report.", rule.Name))
		return
	}

	if p.Report == nil {
		panic("Report function is not set")
	}

	p.Report(analysisName, Diagnostic{
		Name:     rule.Name,
		Severity: rule.Severity,
		Title:    message,
		Detail:   detail,
	})
}

// GetAnalyzerDiagnostics returns all diagnostics reported by the given analyzer.
func (p *Pass) GetAnalyzerDiagnostics(a *Analyzer) []Diagnostic {
	if p.Diagnostics == nil || a == nil {
		return nil
	}
	var result []Diagnostic
	for key, diags := range *p.Diagnostics {
		// Key is the analyzer name when using ReportResult (which all validators use)
		// or a rule name when using Report directly (mainly in tests)
		if key == a.Name || p.AnalyzerRulesMap[key] == a.Name {
			result = append(result, diags...)
		}
	}
	return result
}

// AnalyzerHasErrors returns true if the given analyzer reported any diagnostics with Error severity.
func (p *Pass) AnalyzerHasErrors(a *Analyzer) bool {
	for _, d := range p.GetAnalyzerDiagnostics(a) {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

type Diagnostic struct {
	Severity Severity
	Title    string
	Detail   string
	Context  string `json:"Context,omitempty"`
	Name     string
}

type Diagnostics map[string][]Diagnostic

type Rule struct {
	Name      string
	Disabled  bool
	Severity  Severity
	ReportAll bool
}

type Analyzer struct {
	Name       string
	Requires   []*Analyzer
	Run        func(pass *Pass) (interface{}, error)
	Rules      []*Rule
	ReadmeInfo ReadmeInfo
}

type ReadmeInfo struct {
	Name         string
	Description  string
	Dependencies string
}

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
	AnalyzerName string
	RootDir      string
	CheckParams  CheckParams
	ResultOf     map[*Analyzer]any
	Report       func(string, Diagnostic)
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

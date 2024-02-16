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
)

type Pass struct {
	AnalyzerName  string
	RootDir       string
	SourceCodeDir string
	Checksum      string
	ResultOf      map[*Analyzer]interface{}
	Report        func(string, Diagnostic)
}

func (p *Pass) ReportResult(analysisName string, rule *Rule, message string, detail string) {
	if rule.Disabled {
		logme.Debugln(fmt.Sprintf("Rule %s is disabled. Skipping report.", rule.Name))
		return
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

type Rule struct {
	Name      string
	Disabled  bool
	Severity  Severity
	ReportAll bool
}

type Analyzer struct {
	Name     string
	Requires []*Analyzer
	Run      func(pass *Pass) (interface{}, error)
	Rules    []*Rule
}

package analysis

import (
	"fmt"
)

type Severity string

var (
	Error   Severity = "error"
	Warning Severity = "warning"
	OK      Severity = "ok"
)

type Pass struct {
	AnalyzerName string
	RootDir      string
	ResultOf     map[*Analyzer]interface{}
	Report       func(string, Diagnostic)
}

func (p *Pass) Reportf(analysisName string, rule *Rule, message string, as ...string) {
	if rule.Disabled {
		return
	}

	var is []interface{}
	for _, a := range as {
		is = append(is, a)
	}

	p.Report(analysisName, Diagnostic{
		Name:     rule.Name,
		Severity: rule.Severity,
		Message:  fmt.Sprintf(message, is...),
	})
}

type Diagnostic struct {
	Severity Severity
	Message  string
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

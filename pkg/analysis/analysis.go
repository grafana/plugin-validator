package analysis

import (
	"fmt"
)

type Severity string

var (
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Pass struct {
	RootDir  string
	ResultOf map[*Analyzer]interface{}
	Report   func(Diagnostic)
}

func (p *Pass) Reportf(rule *Rule, message string, as ...string) {
	if rule.Disabled {
		return
	}

	var is []interface{}
	for _, a := range as {
		is = append(is, a)
	}

	p.Report(Diagnostic{
		Severity: rule.Severity,
		Message:  fmt.Sprintf(message, is...),
	})
}

type Diagnostic struct {
	Severity Severity
	Message  string
	Context  string
}

type Rule struct {
	Name     string
	Disabled bool
	Severity Severity
}

type Analyzer struct {
	Name     string
	Requires []*Analyzer
	Run      func(pass *Pass) (interface{}, error)
	Rules    []*Rule
}

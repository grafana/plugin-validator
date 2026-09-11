package analysis

import "testing"

func TestDisabledAnalyzerDoesNotReportIncomplete(t *testing.T) {
	pass := &Pass{Analyzer: &Analyzer{Rules: []*Rule{{Disabled: true}}}, Report: func(string, Diagnostic) { t.Fatal("disabled analyzer reported an execution failure") }}
	pass.ReportIncomplete("failed")
}

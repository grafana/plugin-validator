package testpassinterceptor

import "github.com/grafana/plugin-validator/pkg/analysis"

type TestPassInterceptor struct {
	Diagnostics []*analysis.Diagnostic
}

func (t *TestPassInterceptor) ReportInterceptor() func(string, analysis.Diagnostic) {
	return func(_ string, diagnostic analysis.Diagnostic) {
		t.Diagnostics = append(t.Diagnostics, &diagnostic)
	}
}

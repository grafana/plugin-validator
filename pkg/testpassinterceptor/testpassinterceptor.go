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

func (t *TestPassInterceptor) GetTitles() []string {
	var titles []string
	for _, d := range t.Diagnostics {
		titles = append(titles, d.Title)
	}
	return titles
}
func (t *TestPassInterceptor) GetDetails() []string {
	var details []string
	for _, d := range t.Diagnostics {
		details = append(details, d.Detail)
	}
	return details
}

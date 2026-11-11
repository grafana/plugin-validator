package analysis

// ReportIncomplete records an execution failure rather than a finding about the plugin.
// It is independent of finding severity overrides so a failed scan cannot become a clean result.
func (p *Pass) ReportIncomplete(detail string) {
	// An analyzer with every rule disabled has no requested checks to complete.
	if p.Analyzer != nil && p.Analyzer.AllRulesDisabled() {
		return
	}
	p.Report(p.AnalyzerName, Diagnostic{
		Name:     "scan-incomplete",
		Severity: Error,
		Title:    "Scan incomplete",
		Detail:   detail,
	})
}

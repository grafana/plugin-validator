package analysis

// ReportIncomplete records an execution failure rather than a finding about the plugin.
// It is independent of finding severity overrides so a failed scan cannot become a clean result.
func (p *Pass) ReportIncomplete(detail string) {
	// An analyzer with every rule disabled has no requested checks to complete.
	if p.Analyzer != nil && len(p.Analyzer.Rules) > 0 {
		enabled := false
		for _, rule := range p.Analyzer.Rules {
			enabled = enabled || !rule.Disabled
		}
		if !enabled {
			return
		}
	}
	p.Report(p.AnalyzerName, Diagnostic{
		Name:     "scan-incomplete",
		Severity: Error,
		Title:    "Scan incomplete",
		Detail:   detail,
	})
}

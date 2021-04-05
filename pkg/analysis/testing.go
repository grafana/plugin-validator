package analysis

type TestReporter struct {
	ReportFunc func(d Diagnostic)
	Invoked    bool
}

func (r *TestReporter) Report(d Diagnostic) {
	r.Invoked = true

	if r.ReportFunc != nil {
		r.ReportFunc(d)
	}
}

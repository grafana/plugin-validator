package analysis

type Severity string

var (
	Error   Severity = "error"
	Warning Severity = "warning"
	OK      Severity = "ok"
)

type Pass struct {
	AnalyzerName      string
	RootDir           string
	SourceCodeDir     string
	ResultOf          map[*Analyzer]interface{}
	DependencyResults Results
	Report            func(string, Diagnostic)
}

func (p *Pass) ReportResult(analysisName string, rule *Rule, message string, detail string) {
	if rule.Disabled {
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

type Runnable interface {
	Run(pass *Pass) error

	// GetResult returns the result of the analysis.
	// It is for backwards-compatibility only and will go away once all analyzers have been made static
	// TODO: static analyzers: remove
	GetResult() interface{}
}

type AnalyzerGetter interface {
	GetAnalyzer() *Analyzer
}

type StaticAnalyzer interface {
	Runnable
	AnalyzerGetter
}

type Analyzer struct {
	Name     string
	Requires []*Analyzer
	Run      func(pass *Pass) (interface{}, error)
	Rules    []*Rule

	NewRequires []string
}

func NewAnalyzer(name string) Analyzer {
	return Analyzer{
		Name:     name,
		Requires: []*Analyzer{},
		Rules:    []*Rule{},

		NewRequires: []string{},
	}
}

func (a Analyzer) WithDependencies(deps ...string) Analyzer {
	a.NewRequires = append(a.NewRequires, deps...)
	return a
}

func (a Analyzer) WithRules(rules ...*Rule) Analyzer {
	a.Rules = append(a.Rules, rules...)
	return a
}

func (a Analyzer) GetAnalyzer() *Analyzer {
	return &a
}

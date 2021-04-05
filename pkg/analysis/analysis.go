package analysis

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

type Diagnostic struct {
	Severity Severity
	Message  string
	Context  string
}

type Analyzer struct {
	Name     string
	Requires []*Analyzer
	Run      func(pass *Pass) (interface{}, error)
}

package gosec

type Issue struct {
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	RuleID     string `json:"rule_id"`
	Details    string `json:"details"`
	File       string `json:"file"`
	Code       string `json:"code"`
	Line       string `json:"line"`
}

type Stats struct {
	Files int `json:"files"`
	Lines int `json:"lines"`
	Nosec int `json:"nosec"`
	Found int `json:"found"`
}

type Result struct {
	Issues []Issue `json:"Issues"`
	Stats  Stats   `json:"Stats"`
}

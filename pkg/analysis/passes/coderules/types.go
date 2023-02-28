package coderules

type SemgrepError struct {
}

type SemgrepResult struct {
	Check_id string `json:"check_id"`
	End      struct {
		Col    int `json:"col"`
		Line   int `json:"line"`
		Offset int `json:"offset"`
	} `json:"end"`
	Extra struct {
		Is_ignored bool   `json:"is_ignored"`
		Lines      string `json:"lines"`
		Message    string `json:"message"`
		Severity   string `json:"severity"`
	} `json:"extra"`
	Path  string `json:"path"`
	Start struct {
		Col    int `json:"col"`
		Line   int `json:"line"`
		Offset int `json:"offset"`
	} `json:"start"`
}

type SemgrepResults struct {
	Errors  []SemgrepError  `json:"errors"`
	Results []SemgrepResult `json:"results"`
	Version string          `json:"version"`
}

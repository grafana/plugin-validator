package osvscanner

import "time"

/*
Possible Severity levels returned by osv-scanner:

	CRITICAL
	HIGH
	MODERATE
	LOW
*/

const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityModerate = "MODERATE"
	SeverityLow      = "LOW"
)

// OSVJsonOutput output expected by osv-scanner as a struct
type OSVJsonOutput struct {
	Results []struct {
		Source struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"source"`
		Packages []struct {
			Package struct {
				Name      string `json:"name"`
				Version   string `json:"version"`
				Ecosystem string `json:"ecosystem"`
			} `json:"package"`
			Vulnerabilities []struct {
				SchemaVersion string    `json:"schema_version"`
				ID            string    `json:"id"`
				Modified      time.Time `json:"modified"`
				Published     time.Time `json:"published"`
				Aliases       []string  `json:"aliases"`
				Summary       string    `json:"summary"`
				Details       string    `json:"details"`
				Affected      []struct {
					Package struct {
						Ecosystem string `json:"ecosystem"`
						Name      string `json:"name"`
						Purl      string `json:"purl"`
					} `json:"package"`
					Ranges []struct {
						Type   string `json:"type"`
						Events []struct {
							Introduced string `json:"introduced,omitempty"`
							Fixed      string `json:"fixed,omitempty"`
						} `json:"events"`
					} `json:"ranges"`
					DatabaseSpecific struct {
						Source string `json:"source"`
						URL    string `json:"url,omitempty"`
					} `json:"database_specific"`
				} `json:"affected"`
				References []struct {
					Type string `json:"type"`
					URL  string `json:"url"`
				} `json:"references"`
				DatabaseSpecific struct {
					CweIds         []string `json:"cwe_ids,omitempty"`
					GithubReviewed bool     `json:"github_reviewed,omitempty"`
					Severity       string   `json:"severity,omitempty"`
				} `json:"database_specific"`
			} `json:"vulnerabilities"`
			Groups []struct {
				Ids []string `json:"ids"`
			} `json:"groups"`
		} `json:"packages"`
	} `json:"results"`
}

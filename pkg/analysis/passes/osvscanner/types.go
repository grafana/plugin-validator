package osvscanner

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

var GrafanaPackages = map[string]bool{
	"@grafana/data":    true,
	"@grafana/e2e":     true,
	"@grafana/runtime": true,
	"@grafana/toolkit": true,
	"@grafana/ui":      true,
}

var WhitelistedPackages = map[string]bool{
	// package is a peer dependency of @grafana/plugins-e2e
	"playwright@1.55.0": true,
}

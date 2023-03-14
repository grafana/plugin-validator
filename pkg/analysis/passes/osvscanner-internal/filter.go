package OSVScannerInternal

import (
	"strings"

	"github.com/google/osv-scanner/pkg/models"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/osvscanner-internal/lockfile"
	"github.com/grafana/plugin-validator/pkg/logme"
)

func isFiltered(includeList map[string]bool) bool {
	for packageName := range includeList {
		if GrafanaPackages[packageName] {
			return true
		}
	}
	return false
}

// FilterOSVInternalResults
func FilterOSVInternalResults(source models.VulnerabilityResults, lockFile string) models.VulnerabilityResults {
	// not filtering go.mod yet
	if strings.HasSuffix(lockFile, "go.mod") {
		return source
	}
	var filtered models.VulnerabilityResults
	// this expects a single result, with multiple packages since we are scanning a single file per-run
	if len(source.Results) == 0 {
		// return empty results
		return source
	}
	// parse the lockfile
	parsedPackages, err := lockfile.ParseYarnLock(lockFile)
	if err != nil {
		return source
	}
	// copy the first (and only) result
	filtered.Results = append(filtered.Results, source.Results[0])
	// empty the packages
	filtered.Results[0].Packages = nil
	// iterate over the vulnerabilities and match against our list
	for _, aPackage := range source.Results[0].Packages {
		packageName := aPackage.Package.Name
		includedBy := lockfile.YarnWhyAll(packageName, parsedPackages)
		if !isFiltered(includedBy) {
			logme.DebugFln("not filtered: %s", packageName)
			filtered.Results[0].Packages = append(filtered.Results[0].Packages, aPackage)
		} else {
			logme.DebugFln("excluded by filters: %s", packageName)
		}
	}
	return filtered
}

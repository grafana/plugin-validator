package osvscanner

import (
	"github.com/grafana/plugin-validator/pkg/logme"
	"golang.org/x/exp/maps"
)

func FilterOSVResults(source OSVJsonOutput) OSVJsonOutput {
	// combine all packages
	allPackages := make(map[string]bool, 0)
	maps.Copy(allPackages, CommonPackages)
	maps.Copy(allPackages, GrafanaDataPackages)
	maps.Copy(allPackages, GrafanaE2EPackages)
	maps.Copy(allPackages, GrafanaToolkitPackages)
	maps.Copy(allPackages, GrafanaUIPackages)
	// filter out known packages pulled in by our npm packages
	filtered := filterPackages(allPackages, source)
	return filtered
}

// filterPackages
func filterPackages(filters map[string]bool, source OSVJsonOutput) OSVJsonOutput {
	var filtered OSVJsonOutput
	// this expects a single result, with multiple packages since we are scanning a single file per-run
	if len(source.Results) == 0 {
		// return empty results
		return source
	}
	// copy the first (and only) result
	filtered.Results = append(filtered.Results, source.Results[0])
	// empty the packages
	filtered.Results[0].Packages = nil
	// process
	for _, aPackage := range source.Results[0].Packages {
		packageName := aPackage.Package.Name
		if !filters[packageName] {
			logme.DebugFln("not filtered: %s", packageName)
			filtered.Results[0].Packages = append(filtered.Results[0].Packages, aPackage)
		} else {
			logme.DebugFln("excluded by filters: %s", packageName)
		}
	}
	return filtered
}

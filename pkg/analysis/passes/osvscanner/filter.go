package osvscanner

import "golang.org/x/exp/maps"

func FilterOSVResults(source OSVJsonOutput) OSVJsonOutput {
	// combine all packages
	allPackages := make(map[string]bool, 0)
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
	for _, result := range source.Results {
		for _, aPackage := range result.Packages {
			packageName := aPackage.Package.Name
			if !filters[packageName] {
				filtered.Results = append(filtered.Results, result)
			}
		}
	}
	return filtered
}

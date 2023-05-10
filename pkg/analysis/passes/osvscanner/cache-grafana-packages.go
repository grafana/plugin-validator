package osvscanner

import (
	"sort"

	"github.com/grafana/plugin-validator/pkg/analysis/passes/osvscanner/lockfile"
)

func packageExists(name string, items []lockfile.PackageDetails) bool {
	for _, aPackage := range items {
		if aPackage.Name == name {
			return true
		}
	}
	return false
}

func IncludedByGrafanaPackage(packageName string, cache []lockfile.PackageFlattened) (bool, string) {
	for _, item := range cache {
		for _, dependency := range item.Dependencies {
			if dependency.Package.Name == packageName {
				return true, item.Name
			}
		}
	}
	return false, ""
}

func CacheGrafanaPackages(allPackages []lockfile.PackageDetails) ([]lockfile.PackageFlattened, error) {
	cache := make([]lockfile.PackageFlattened, 0)
	for grafanaPackage := range GrafanaPackages {
		// check if the package is in the list parsed
		if packageExists(grafanaPackage, allPackages) {
			// get dependencies of grafanaPackage
			expanded, err := lockfile.ExpandPackage(grafanaPackage, allPackages)
			if err != nil {
				// failed to parse
				return nil, err
			}
			cache = append(cache, *expanded)
		}
	}
	// sort cache
	sort.SliceStable(cache, func(i, j int) bool {
		return cache[i].Name < cache[j].Name
	})

	return cache, nil
}

package osvscanner

import (
	"sort"

	"github.com/grafana/plugin-validator/pkg/analysis/passes/osvscanner/lockfile"
)

func IncludedByGrafanaPackage(
	packageName string,
	cache []lockfile.PackageFlattened,
) (bool, string) {
	for _, item := range cache {
		for _, dependency := range item.Dependencies {
			if dependency.Package.Name == packageName {
				return true, item.Name
			}
		}
	}
	return false, ""
}

func CacheGrafanaPackages(
	allPackages []lockfile.PackageDetails,
) ([]lockfile.PackageFlattened, error) {
	cache := make([]lockfile.PackageFlattened, 0)
	processedNames := make(map[string]bool)

	for _, pkg := range allPackages {
		if GrafanaPackages[pkg.Name] && !processedNames[pkg.Name] {
			processedNames[pkg.Name] = true

			allVersions := make([]lockfile.PackageDetails, 0)
			for _, p := range allPackages {
				if p.Name == pkg.Name {
					allVersions = append(allVersions, p)
				}
			}

			// should never happen
			if len(allVersions) == 0 {
				continue
			}

			// create a custom flattened package
			// with all dependencies for all versions of this package
			merged := lockfile.PackageFlattened{
				Name:    pkg.Name,
				Version: allVersions[0].Version,
			}

			for _, version := range allVersions {
				for _, dep := range version.Dependencies {
					exists := false
					for _, existingDep := range merged.Dependencies {
						if existingDep.Package.Name == dep.Name {
							exists = true
							break
						}
					}
					if !exists {
						state := lockfile.DependencyState{
							Package:   dep,
							Processed: false,
						}
						merged.Dependencies = append(merged.Dependencies, state)
					}
				}
			}

			// find all transitive dependencies of this package
			// aka deps of deps of deps of deps to the infinite
			expandDependenciesRecursively(&merged, allPackages)

			cache = append(cache, merged)
		}
	}

	sort.SliceStable(cache, func(i, j int) bool {
		return cache[i].Name < cache[j].Name
	})

	return cache, nil
}

func expandDependenciesRecursively(
	topLevel *lockfile.PackageFlattened,
	allPackages []lockfile.PackageDetails,
) {
	for i := 0; i < len(topLevel.Dependencies); i++ {
		if topLevel.Dependencies[i].Processed {
			continue
		}

		topLevel.Dependencies[i].Processed = true
		depName := topLevel.Dependencies[i].Package.Name

		var depPackage *lockfile.PackageDetails
		for j := range allPackages {
			if allPackages[j].Name == depName {
				depPackage = &allPackages[j]
				break
			}
		}

		if depPackage == nil {
			continue
		}

		for _, subDep := range depPackage.Dependencies {
			exists := false
			for k := range topLevel.Dependencies {
				if topLevel.Dependencies[k].Package.Name == subDep.Name {
					exists = true
					break
				}
			}

			if !exists {
				state := lockfile.DependencyState{
					Package:   subDep,
					Processed: false,
				}
				topLevel.Dependencies = append(topLevel.Dependencies, state)
			}
		}
	}
}

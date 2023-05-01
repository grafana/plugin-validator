package lockfile

import (
	"fmt"
	"sort"
)

// getPackageDetails
// returns a specific package from the array, nil if it is not found
func getPackageDetails(locate string, packages []PackageDetails) *PackageDetails {
	// locate the package
	for _, aPackage := range packages {
		if aPackage.Name == locate {
			return &aPackage
		}
	}
	return nil
}

// getFlattenedPackageDependencies
func getFlattenedPackageDependencies(locate string, packages []PackageDetails) (*PackageFlattened, error) {
	aPackage := getPackageDetails(locate, packages)

	if aPackage == nil {
		// not in our lock file
		return nil, fmt.Errorf("package not found: %s", locate)
	}
	flattened := PackageFlattened{
		Name:    aPackage.Name,
		Version: aPackage.Version,
	}

	for _, aPackage := range packages {
		if aPackage.Name == locate {
			for _, aDependency := range aPackage.Dependencies {
				state := DependencyState{
					Package:   aDependency,
					Processed: false,
				}
				flattened.Dependencies = append(flattened.Dependencies, state)
			}
			break
		}
	}
	return &flattened, nil
}

func dependencyExists(item DependencyState, items []DependencyState) bool {
	for i := range items {
		if items[i].Package.Name == item.Package.Name {
			return true
		}
	}
	return false
}

func deduplicate(source *PackageFlattened, nested *PackageFlattened) {
	for _, aDependency := range nested.Dependencies {
		matched := false
		for i := range source.Dependencies {
			if aDependency.Package.Name == source.Dependencies[i].Package.Name {
				// this is a duplicate
				matched = true
				break
			}
		}
		if !matched {
			// this is new, append to source
			source.Dependencies = append(source.Dependencies, aDependency)
		}
	}
}

func deepExpand(topLevelPackage *PackageFlattened, expandPackage *PackageFlattened, packages []PackageDetails) {
	if expandPackage == nil {
		expandPackage, _ = getFlattenedPackageDependencies(topLevelPackage.Name, packages)
	}
	outerPackage, _ := getFlattenedPackageDependencies(expandPackage.Name, packages)
	// remove duplicates
	deduplicate(topLevelPackage, outerPackage)
	// scan primary dependency list for unprocessed entries
	for primaryPackageDependencyIndex, primaryPackageDependency := range topLevelPackage.Dependencies {
		if !primaryPackageDependency.Processed {
			// set processed to true (prevents re-scanning this package)
			topLevelPackage.Dependencies[primaryPackageDependencyIndex].Processed = true
			// get the dependencies for the new package entry
			packageToProcess, _ := getFlattenedPackageDependencies(primaryPackageDependency.Package.Name, packages)
			addedPackages := false
			for _, subDependency := range packageToProcess.Dependencies {
				// if subDependency is not in topLevelPackage dependencies, add it, then expand again
				if !dependencyExists(subDependency, topLevelPackage.Dependencies) {
					// add to primary
					topLevelPackage.Dependencies = append(topLevelPackage.Dependencies, subDependency)
					addedPackages = true
				}
			}
			if addedPackages {
				deepExpand(topLevelPackage, packageToProcess, packages)
			}
		}
	}
}

func ExpandPackage(packageName string, packages []PackageDetails) (*PackageFlattened, error) {
	// get dependencies for specified package
	primaryPackageFlattened, err := getFlattenedPackageDependencies(packageName, packages)
	// recursively expand dependencies
	if err != nil {
		return nil, err
	}
	deepExpand(primaryPackageFlattened, nil, packages)
	// sort for easier debugging
	sort.Slice(primaryPackageFlattened.Dependencies, func(i, j int) bool {
		return primaryPackageFlattened.Dependencies[i].Package.Name < primaryPackageFlattened.Dependencies[j].Package.Name
	})
	return primaryPackageFlattened, nil
}

package lockfile

import "errors"

// getPackageDetails
// Returns a specific package from the array, and error if it is not found
func getPackageDetails(locate string, packages []PackageDetails) (*PackageDetails, error) {
	// locate the package
	for _, aPackage := range packages {
		if aPackage.Name == locate {
			return &aPackage, nil
		}
	}
	return nil, errors.New("not found")
}

// YarnWhyAll
// Returns list of packages that caused the specified package to be pulled in
// NOTE: this is a full list, without the hierarchy included (for speed)
func YarnWhyAll(locate string, packages []PackageDetails) map[string]bool {
	includedBy := make(map[string]bool)

	_, err := getPackageDetails(locate, packages)
	if err != nil {
		// package not found, return empty
		return includedBy
	}

	// check if the package is a dependency anywhere in the lock file
	for _, aPackage := range packages {
		for _, dependency := range aPackage.Dependencies {
			if dependency.Name == locate {
				// found it
				includedBy[aPackage.Name] = true
				// check who included this package too
				nestedPackages := YarnWhyAll(aPackage.Name, packages)
				for nestedPackage := range nestedPackages {
					includedBy[nestedPackage] = true
				}
			}
		}
	}
	return includedBy
}

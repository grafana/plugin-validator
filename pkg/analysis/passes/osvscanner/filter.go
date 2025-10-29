package osvscanner

import (
	"path"
	"strings"

	"github.com/google/osv-scanner/v2/pkg/models"

	"github.com/grafana/plugin-validator/pkg/analysis/passes/osvscanner/lockfile"
	"github.com/grafana/plugin-validator/pkg/logme"
)

func FilterOSVResults(source models.VulnerabilityResults, lockFile string) models.VulnerabilityResults {
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
	lockFileType := path.Base(lockFile)
	// parse the lockfile
	var parsedPackages []lockfile.PackageDetails
	var parseError error
	switch lockFileType {
	case "yarn.lock":
		parsedPackages, parseError = lockfile.ParseYarnLock(lockFile)
	case "package-lock.json":
		parsedPackages, parseError = lockfile.ParseNpmLock(lockFile)
	case "pnpm-lock.yaml":
		parsedPackages, parseError = lockfile.ParsePnpmLock(lockFile)
	}
	if parseError != nil {
		return source
	}
	// copy the first (and only) result
	filtered.Results = append(filtered.Results, source.Results[0])
	// empty the packages
	filtered.Results[0].Packages = nil
	cachedPackages, err := CacheGrafanaPackages(parsedPackages)
	if err != nil {
		// cache error
		logme.Errorln("cache failure", err)
		return filtered
	}
	// iterate over the vulnerabilities and match against our list
	for _, aPackage := range source.Results[0].Packages {
		packageName := aPackage.Package.Name
		packageVersion := aPackage.Package.Version
		packageWithVersion := packageName + "@" + packageVersion

		// Check if package is whitelisted
		if WhitelistedPackages[packageWithVersion] {
			logme.DebugFln("excluded by whitelist: %s", packageWithVersion)
			continue
		}

		cacheHit, includedBy := IncludedByGrafanaPackage(packageName, cachedPackages)
		if !cacheHit {
			//logme.DebugFln("not filtered: %s", packageName)
			filtered.Results[0].Packages = append(filtered.Results[0].Packages, aPackage)
		} else {
			logme.DebugFln("excluded by filter (%s): %s", includedBy, packageName)
		}
	}
	return filtered
}

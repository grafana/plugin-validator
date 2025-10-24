package osvscanner

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis/passes/osvscanner/lockfile"
)

func TestCacheGrafanaPackages(t *testing.T) {
	aLockfile := filepath.Join("testdata", "node", "circular-yarn", "yarn.lock")
	packages, err := lockfile.ParseYarnLock(aLockfile)
	require.NoError(t, err)
	cachedPackages, err := CacheGrafanaPackages(packages)
	require.NoError(t, err)
	require.Equal(t, 4, len(cachedPackages))
	require.Equal(t, "@grafana/data", cachedPackages[0].Name)
	require.Equal(t, "@grafana/runtime", cachedPackages[1].Name)
}

func TestCacheHitMiss(t *testing.T) {
	aLockfile := filepath.Join("testdata", "node", "circular-yarn", "yarn.lock")
	packages, err := lockfile.ParseYarnLock(aLockfile)
	require.NoError(t, err)
	cachedPackages, err := CacheGrafanaPackages(packages)
	require.NoError(t, err)
	cacheHit, includedBy := IncludedByGrafanaPackage("moment", cachedPackages)
	require.True(t, cacheHit)
	require.Equal(t, "@grafana/data", includedBy)
	cacheMiss, _ := IncludedByGrafanaPackage("notmoment", cachedPackages)
	require.False(t, cacheMiss)
}

func TestCacheGrafanaPackagesMultipleVersionsNPM(t *testing.T) {
	t.Parallel()

	aLockfile := filepath.Join("testdata", "node", "multi-version-npm", "package-lock.json")
	packages, err := lockfile.ParseNpmLock(aLockfile)
	require.NoError(t, err)

	cachedPackages, err := CacheGrafanaPackages(packages)
	require.NoError(t, err)

	// This test verifies the fix: when multiple versions of @grafana/data exist (9.5.1 with dompurify and 9.3.8 without), both versions should be cached so dompurify is found
	var foundGrafanaData bool
	var hasDompurify bool
	for _, pkg := range cachedPackages {
		if pkg.Name == "@grafana/data" {
			foundGrafanaData = true
			for _, dep := range pkg.Dependencies {
				if dep.Package.Name == "dompurify" {
					hasDompurify = true
					break
				}
			}
			break
		}
	}

	require.True(t, foundGrafanaData, "@grafana/data should be in the cache")
	require.True(t, hasDompurify, "dompurify should be in @grafana/data dependencies")

	cacheHit, includedBy := IncludedByGrafanaPackage("dompurify", cachedPackages)
	require.True(t, cacheHit, "dompurify should be found in the cache")
	require.Equal(t, "@grafana/data", includedBy, "dompurify should be included by @grafana/data")
}

func TestCacheGrafanaPackagesSingleVersionNPM(t *testing.T) {
	t.Parallel()

	aLockfile := filepath.Join("testdata", "node", "critical-npm", "package-lock.json")
	packages, err := lockfile.ParseNpmLock(aLockfile)
	require.NoError(t, err)

	cachedPackages, err := CacheGrafanaPackages(packages)
	require.NoError(t, err)

	require.NotEmpty(t, cachedPackages, "cache should have entries")

	cacheHit, includedBy := IncludedByGrafanaPackage("moment", cachedPackages)
	require.True(t, cacheHit, "moment should be found in the cache")
	require.Equal(t, "@grafana/data", includedBy, "moment should be included by @grafana/data")
}

func TestCacheGrafanaPackagesYarnLock(t *testing.T) {
	t.Parallel()

	aLockfile := filepath.Join("testdata", "node", "circular-yarn", "yarn.lock")
	packages, err := lockfile.ParseYarnLock(aLockfile)
	require.NoError(t, err)

	cachedPackages, err := CacheGrafanaPackages(packages)
	require.NoError(t, err)

	grafanaPackageCount := 0
	grafanaPackages := make(map[string]bool)
	for _, pkg := range cachedPackages {
		if pkg.Name == "@grafana/data" || pkg.Name == "@grafana/runtime" || pkg.Name == "@grafana/toolkit" || pkg.Name == "@grafana/ui" {
			grafanaPackageCount++
			grafanaPackages[pkg.Name] = true
		}
	}

	require.GreaterOrEqual(t, grafanaPackageCount, 2, "should have at least 2 different @grafana/* packages in cache")
	require.True(t, grafanaPackages["@grafana/data"], "@grafana/data should be in cache")
	require.True(t, grafanaPackages["@grafana/runtime"], "@grafana/runtime should be in cache")

	cacheHit, includedBy := IncludedByGrafanaPackage("moment", cachedPackages)
	require.True(t, cacheHit, "moment should be found in the cache")
	require.Equal(t, "@grafana/data", includedBy, "moment should be included by @grafana/data")
}

func TestNonGrafanaDependencyVulnerabilitiesAreNotFiltered(t *testing.T) {
	t.Parallel()

	aLockfile := filepath.Join("testdata", "node", "multi-version-npm", "package-lock.json")
	packages, err := lockfile.ParseNpmLock(aLockfile)
	require.NoError(t, err)

	cachedPackages, err := CacheGrafanaPackages(packages)
	require.NoError(t, err)

	bodyParserInCache, _ := IncludedByGrafanaPackage("body-parser", cachedPackages)
	require.False(t, bodyParserInCache, "body-parser should NOT be included by any Grafana package")

	dompurifyInCache, includedByDompurify := IncludedByGrafanaPackage("dompurify", cachedPackages)
	require.True(t, dompurifyInCache, "dompurify should be included by a Grafana package")
	require.Equal(t, "@grafana/data", includedByDompurify, "dompurify should be included by @grafana/data")
}

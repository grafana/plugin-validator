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

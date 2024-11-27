package lockfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandPackageCircular(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "circular-yarn", "yarn.lock")
	packages, err := ParseYarnLock(aLockfile)
	require.NoError(t, err)

	// this one would cause an infinite loop
	expandedCircular, err := ExpandPackage("@jest/test-sequencer", packages)
	require.NoError(t, err)
	require.Len(t, expandedCircular.Dependencies, 416)
}

func TestExpandGrafanaPackagesFromYarn(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "circular-yarn", "yarn.lock")
	packages, err := ParseYarnLock(aLockfile)
	require.NoError(t, err)

	expandedGrafanaData, err := ExpandPackage("@grafana/data", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaData.Dependencies, 54)

	expandedGrafanaE2E, err := ExpandPackage("@grafana/e2e", packages)
	require.Error(t, err)
	require.Nil(t, expandedGrafanaE2E)

	expandedGrafanaRuntime, err := ExpandPackage("@grafana/runtime", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaRuntime.Dependencies, 352)

	expandedGrafanaToolkit, err := ExpandPackage("@grafana/toolkit", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaToolkit.Dependencies, 1307)

	expandedGrafanaUI, err := ExpandPackage("@grafana/ui", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaUI.Dependencies, 350)
}

func TestExpandGrafanaPackagesFromNPM(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "critical-npm", "package-lock.json")
	packages, err := ParseNpmLock(aLockfile)
	require.NoError(t, err)

	expandedGrafanaData, err := ExpandPackage("@grafana/data", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaData.Dependencies, 61)

	expandedGrafanaToolkit, err := ExpandPackage("@grafana/toolkit", packages)
	require.NotNil(t, err)
	require.Equal(t, err.Error(), "package not found: @grafana/toolkit")
	require.Nil(t, expandedGrafanaToolkit)

	expandedGrafanaE2E, err := ExpandPackage("@grafana/e2e", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaE2E.Dependencies, 422)

	expandedGrafanaRuntime, err := ExpandPackage("@grafana/runtime", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaRuntime.Dependencies, 404)

	_, err = ExpandPackage("@grafana/toolkit", packages)
	require.Error(t, err)

	expandedGrafanaUI, err := ExpandPackage("@grafana/ui", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaUI.Dependencies, 389)
}

func TestExpandGrafanaPackagesFromPnpm(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "critical-pnpm", "pnpm-lock.yaml")
	packages, err := ParsePnpmLock(aLockfile)
	require.NoError(t, err)

	expandedGrafanaData, err := ExpandPackage("@grafana/data", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaData.Dependencies, 64)

	expandedGrafanaToolkit, err := ExpandPackage("@grafana/toolkit", packages)
	require.NotNil(t, err)
	require.Equal(t, err.Error(), "package not found: @grafana/toolkit")
	require.Nil(t, expandedGrafanaToolkit)

	expandedGrafanaE2E, err := ExpandPackage("@grafana/e2e", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaE2E.Dependencies, 485)

	expandedGrafanaRuntime, err := ExpandPackage("@grafana/runtime", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaRuntime.Dependencies, 439)

	_, err = ExpandPackage("@grafana/toolkit", packages)
	require.Error(t, err)

	expandedGrafanaUI, err := ExpandPackage("@grafana/ui", packages)
	require.NoError(t, err)
	require.Len(t, expandedGrafanaUI.Dependencies, 424)
}

package lockfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePnpmLock_Dependencies(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "critical-pnpm", "pnpm-lock.yaml")
	packages, err := ParsePnpmLock(aLockfile)
	require.NoError(t, err)
	require.Len(t, packages[4].Dependencies, 15)
}

// Regression test for https://github.com/grafana/plugin-validator/pull/282
// Corrupted lockfiles with malformed package keys should be handled gracefully.
func TestParsePnpmLock_CorruptedLockfile(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "corrupted-pnpm", "pnpm-lock.yaml")
	packages, err := ParsePnpmLock(aLockfile)
	require.NoError(t, err)
	// Only the 2 valid packages should be parsed, corrupted entries should be skipped
	require.Len(t, packages, 2)
	// Packages are sorted alphabetically by name
	require.Equal(t, "another-valid", packages[0].Name)
	require.Equal(t, "2.0.0", packages[0].Version)
	require.Equal(t, "valid-pkg", packages[1].Name)
	require.Equal(t, "1.0.0", packages[1].Version)
}

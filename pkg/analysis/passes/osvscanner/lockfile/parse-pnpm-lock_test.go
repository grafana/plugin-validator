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

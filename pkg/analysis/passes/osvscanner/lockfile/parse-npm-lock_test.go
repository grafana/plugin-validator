package lockfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNPMLock_Dependencies(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "critical-npm", "package-lock.json")
	packages, err := ParseNpmLock(aLockfile)
	require.NoError(t, err)
	require.Len(t, packages[4].Dependencies, 15)
}

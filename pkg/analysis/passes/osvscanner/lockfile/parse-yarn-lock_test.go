package lockfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseYarnLock_v2_Dependencies(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "critical-yarn", "yarn.lock")
	packages, err := ParseYarnLock(aLockfile)
	require.NoError(t, err)
	require.Len(t, packages[0].Dependencies, 2)
	// @grafana/data
	require.Len(t, packages[157].Dependencies, 20)
	// @grafana/toolkit
	require.Len(t, packages[164].Dependencies, 84)
	// @grafana/ui
	require.Len(t, packages[166].Dependencies, 64)
}

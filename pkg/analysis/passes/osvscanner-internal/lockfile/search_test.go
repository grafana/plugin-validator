package lockfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYarnWhyAll(t *testing.T) {
	t.Parallel()
	aLockfile := filepath.Join("..", "testdata", "node", "critical", "yarn.lock")
	packages, err := ParseYarnLock(aLockfile)
	require.NoError(t, err)

	includedBy := YarnWhyAll("@grafana/toolkit", packages)
	require.Len(t, includedBy, 1)

	includedBy = YarnWhyAll("moment-timezone", packages)
	require.Len(t, includedBy, 5)
	require.Equal(t, includedBy["@grafana/data"], true)

	includedBy = YarnWhyAll("@babel/plugin-syntax-bigint", packages)
	require.Len(t, includedBy, 16)
	require.Equal(t, includedBy["babel-preset-current-node-syntax"], true)

	includedBy = YarnWhyAll("babel-preset-current-node-syntax", packages)
	require.Len(t, includedBy, 15)
	require.Equal(t, includedBy["babel-preset-jest"], true)

	includedBy = YarnWhyAll("babel-preset-current-node-syntax", packages)
	require.Len(t, includedBy, 15)
	require.Equal(t, includedBy["@grafana/toolkit"], true)
}

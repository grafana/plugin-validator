package genreadme

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const readmeFileName = "README.md"

// TestGenReadme ensures that the README is up-to-date
func TestGenReadme(t *testing.T) {
	// Open existing README
	dir := "."
	for {
		if _, err := os.Stat(filepath.Join(dir, readmeFileName)); err == nil {
			break
		}
		dir = filepath.Join(dir, "..")
		if dir == string(os.PathSeparator) {
			// root reached
			require.FailNow(t, "README.md not found")
		}
	}
	readme, err := os.Open(filepath.Join(dir, readmeFileName))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, readme.Close())
	})

	// Re-generate the readme in memory
	gen, err := Generate(readme)
	require.NoError(t, err)
	require.NotEmpty(t, gen)

	// Seek and read the existing readme again (from disk)
	_, err = readme.Seek(0, io.SeekStart)
	require.NoError(t, err)
	existingReadme, err := io.ReadAll(readme)
	require.NoError(t, err)
	require.NotEmpty(t, existingReadme)

	// Ensure the generated readme matches the existing one
	require.Equal(
		t,
		string(existingReadme), gen,
		"README.md is not up-to-date. Run `mage gen:readme` to update it.",
	)
}

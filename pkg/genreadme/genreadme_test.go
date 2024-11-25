package genreadme

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis/passes"
)

const readmeFileName = "README.md"

var flagGenerate = flag.Bool("generate", false, "generate readme file")

func TestGenReadme(t *testing.T) {
	t.Run("generate", func(t *testing.T) {
		if !*flagGenerate {
			t.Skip("skipping generating readme file")
			return
		}
		readmeFn, err := findReadme()
		require.NoError(t, err)
		require.NoError(t, generateToFile(readmeFn))
	})

	t.Run("check", func(t *testing.T) {
		// Find and open existing README
		fn, err := findReadme()
		require.NoError(t, err)
		readme, err := os.Open(fn)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, readme.Close())
		})

		// Re-generate the readme in memory
		gen, err := generate(readme)
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
	})

	t.Run("all analyzers should have readme data", func(t *testing.T) {
		for _, analyzer := range passes.Analyzers {
			assert.Falsef(
				t,
				analyzer.ReadmeInfo.Name == "" && analyzer.ReadmeInfo.Description == "",
				"analyzer %q does not have README data", analyzer.Name,
			)
		}
	})
}

func generateToFile(fn string) error {
	// Read existing README
	f, err := os.Open(fn)
	if err != nil {
		return fmt.Errorf("open readme: %w", err)
	}
	var closed bool
	defer func() {
		if closed {
			return
		}
		_ = f.Close()
	}()
	generatedReadme, err := generate(f)
	if err != nil {
		return fmt.Errorf("generate new readme: %w", err)
	}
	closed = true
	if err = f.Close(); err != nil {
		return fmt.Errorf("close readme: %w", err)
	}

	// Overwrite the README
	outF, err := os.Create(fn)
	if err != nil {
		return fmt.Errorf("create new readme: %w", err)
	}
	if _, err := outF.WriteString(generatedReadme); err != nil {
		return fmt.Errorf("write new readme: %w", err)
	}
	if err := outF.Close(); err != nil {
		return fmt.Errorf("close new readme: %w", err)
	}
	return nil
}

func findReadme() (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("get absolute path: %w", err)
	}
	for {
		fn := filepath.Join(dir, readmeFileName)
		if _, err := os.Stat(fn); err == nil {
			return fn, nil
		}
		newDir := filepath.Dir(dir)
		if newDir == string(os.PathSeparator) || newDir == dir {
			// root reached
			return "", errors.New("README.md not found")
		}
		dir = newDir
	}
}

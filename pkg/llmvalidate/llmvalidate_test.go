package llmvalidate

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPromptsForCodepathWithSubPaths(t *testing.T) {
	codePath := filepath.Join("testdata", "testplugin")

	prompts, err := getPromptContentForCode(codePath, []string{"src", "pkg"})

	require.NoError(t, err)
	require.Len(t, prompts, 2)
}

func TestGetPromptsForCodepathWithoutSubPaths(t *testing.T) {
	codePath := filepath.Join("testdata", "testplugin")

	prompts, err := getPromptContentForCode(codePath, []string{"."})

	require.NoError(t, err)
	require.Len(t, prompts, 4)
}

// should not fail if subpath does not exist
func TestGetPromptsForCodepathWithNonExistingSubPaths(t *testing.T) {
	codePath := filepath.Join("testdata", "testplugin")

	prompts, err := getPromptContentForCode(codePath, []string{"src", "does-not-exist"})

	require.NoError(t, err)
	require.Len(t, prompts, 1)
}

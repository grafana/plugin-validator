package llmclient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create a nested directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src", "nested"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "lib.go"), []byte("package src\nfunc Hello() {}\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "nested", "deep.go"), []byte("package nested\n"), 0644))

	return dir
}

func TestValidatePath_WithinRepo(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	tests := []struct {
		name string
		path string
	}{
		{name: "root dot", path: "."},
		{name: "file at root", path: "main.go"},
		{name: "nested file", path: "src/lib.go"},
		{name: "deeply nested", path: "src/nested/deep.go"},
		{name: "directory", path: "src"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := executor.validatePath(tt.path)
			require.NoError(t, err)
			require.NotEmpty(t, absPath)
		})
	}
}

func TestValidatePath_TraversalBlocked(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	tests := []struct {
		name string
		path string
	}{
		{name: "parent directory", path: ".."},
		{name: "parent with path", path: "../etc/passwd"},
		{name: "double parent", path: "../../etc/passwd"},
		{name: "dot dot in middle", path: "src/../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := executor.validatePath(tt.path)
			require.Empty(t, absPath)
			require.ErrorContains(t, err, "outside the repository")
		})
	}
}

func TestValidatePath_SymlinkEscape(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	// Create a file outside the repo that the symlink will point to.
	externalDir := t.TempDir()
	externalFile := filepath.Join(externalDir, "secret.txt")
	require.NoError(t, os.WriteFile(externalFile, []byte("secret"), 0644))

	// Place a symlink inside the repo pointing to the external file.
	symlinkPath := filepath.Join(dir, "escape.txt")
	require.NoError(t, os.Symlink(externalFile, symlinkPath))

	absPath, err := executor.validatePath("escape.txt")
	require.Empty(t, absPath)
	require.ErrorContains(t, err, "outside the repository")
}

func TestReadFile_PathTraversal(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	_, err := executor.readFile(map[string]interface{}{
		"path": "../../etc/passwd",
	})
	require.ErrorContains(t, err, "outside the repository")
}

func TestReadFile_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	executor := newToolExecutor(dir)

	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "binary file with invalid utf-8",
			content: []byte{0xff, 0xfe, 0x00, 0x01, 0x02, 0x03},
		},
		{
			name:    "png image magic bytes",
			content: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(filepath.Join(dir, "binary.bin"), tt.content, 0644))
			_, err := executor.readFile(map[string]interface{}{"path": "binary.bin"})
			require.ErrorContains(t, err, "not a text file")
		})
	}
}

func TestReadFile_MaxFileSize(t *testing.T) {
	dir := t.TempDir()
	executor := newToolExecutor(dir)

	// Create a file that exceeds the max size
	bigContent := strings.Repeat("x", maxFileSize+1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.txt"), []byte(bigContent), 0644))

	_, err := executor.readFile(map[string]interface{}{
		"path": "big.txt",
	})
	require.ErrorContains(t, err, "file is too large")

	// A file under the limit should work fine
	smallContent := "hello world"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "small.txt"), []byte(smallContent), 0644))

	result, err := executor.readFile(map[string]interface{}{
		"path": "small.txt",
	})
	require.NoError(t, err)
	require.Equal(t, smallContent, result)
}

func TestListDirectory_PathTraversal(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	_, err := executor.listDirectory(map[string]interface{}{
		"path": "../",
	})
	require.ErrorContains(t, err, "outside the repository")
}

func TestGrep_PathTraversal(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	_, err := executor.grep(map[string]interface{}{
		"pattern": "root",
		"path":    "../../etc",
	})
	require.ErrorContains(t, err, "outside the repository")
}

func TestGrep_FlagInjection(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	// Pattern starting with - should not be interpreted as a flag
	// thanks to the -- separator
	result, err := executor.grep(map[string]interface{}{
		"pattern": "-rn",
		"path":    ".",
	})
	// Should not error with "invalid option", should just return no matches
	require.NoError(t, err)
	require.NotContains(t, result, "invalid option")
}

func TestGrep_ExitCodes(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	// Exit code 1: pattern not found — not an error, just no results.
	result, err := executor.grep(map[string]interface{}{
		"pattern": "this_pattern_will_never_match_xyz123",
		"path":    ".",
	})
	require.NoError(t, err)
	require.Equal(t, "No matches found", result)

	// Exit code 2: invalid regex — should be a real error, not silently swallowed.
	_, err = executor.grep(map[string]interface{}{
		"pattern": "[invalid",
		"path":    ".",
	})
	require.Error(t, err)
}

func TestGit_BlockedFlags(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	tests := []struct {
		name string
		args string
	}{
		{name: "exec flag", args: "log --exec=malicious"},
		{name: "ext-diff flag", args: "diff --ext-diff"},
		{name: "upload-pack", args: "fetch --upload-pack=evil"},
		{name: "config flag", args: "log -c core.pager=evil"},
		{name: "config equals", args: "log --config=evil"},
		{name: "config concatenated", args: "log -ccore.pager=evil"},
		{name: "config concatenated ssh", args: "log -ccore.sshCommand=evil"},
		{name: "git prefix exec flag", args: "git log --exec=malicious"},
		{name: "git prefix ext-diff", args: "git diff --ext-diff"},
		{name: "git prefix upload-pack", args: "git fetch --upload-pack=evil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.git(map[string]interface{}{
				"args": tt.args,
			})
			require.ErrorContains(t, err, "not allowed for security reasons")
		})
	}
}

func TestGit_StripGitPrefix(t *testing.T) {
	dir := t.TempDir()
	executor := newToolExecutor(dir)

	// "git status" should be treated the same as "status" — not blocked by allowlist
	_, err := executor.git(map[string]interface{}{
		"args": "git status",
	})
	if err != nil {
		require.NotContains(t, err.Error(), "is not allowed")
	}

	// "git push" should still be blocked
	_, err = executor.git(map[string]interface{}{
		"args": "git push",
	})
	require.ErrorContains(t, err, "is not allowed")

	// bare "git" with nothing after should error
	_, err = executor.git(map[string]interface{}{
		"args": "git",
	})
	require.ErrorContains(t, err, "empty git command")
}

func TestGit_BlockedSubcommand(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	_, err := executor.git(map[string]interface{}{
		"args": "push origin main",
	})
	require.ErrorContains(t, err, "not allowed")
}

func TestGit_AllowedSubcommands(t *testing.T) {
	for subcmd := range allowedGitSubcommands {
		t.Run(subcmd, func(t *testing.T) {
			// Just verify the subcommand is not rejected by our allowlist check.
			// The git command itself may fail (not a real repo) but the error
			// should come from git, not from our validation.
			dir := t.TempDir()
			executor := newToolExecutor(dir)

			_, err := executor.git(map[string]interface{}{
				"args": subcmd,
			})
			if err != nil {
				require.NotContains(t, err.Error(), "is not allowed")
			}
		})
	}
}

func TestExecute(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	t.Run("success", func(t *testing.T) {
		args, _ := json.Marshal(map[string]string{"path": "main.go"})
		result, err := executor.execute("read_file", string(args))
		require.NoError(t, err)
		require.Contains(t, result, "package main")
	})

	t.Run("unknown tool returns error", func(t *testing.T) {
		_, err := executor.execute("nonexistent_tool", "{}")
		require.ErrorContains(t, err, "unknown tool")
	})
}

package llmclient

import (
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
		name    string
		path    string
		wantErr bool
	}{
		{name: "root dot", path: ".", wantErr: false},
		{name: "file at root", path: "main.go", wantErr: false},
		{name: "nested file", path: "src/lib.go", wantErr: false},
		{name: "deeply nested", path: "src/nested/deep.go", wantErr: false},
		{name: "directory", path: "src", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, errMsg := executor.validatePath(tt.path)
			require.Empty(t, errMsg)
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
			absPath, errMsg := executor.validatePath(tt.path)
			require.Empty(t, absPath)
			require.Contains(t, errMsg, "outside the repository")
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

	absPath, errMsg := executor.validatePath("escape.txt")
	require.Empty(t, absPath)
	require.Contains(t, errMsg, "outside the repository")
}

func TestReadFile_PathTraversal(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	result := executor.readFile(map[string]interface{}{
		"path": "../../etc/passwd",
	})
	require.Contains(t, result, "outside the repository")
}

func TestReadFile_MaxFileSize(t *testing.T) {
	dir := t.TempDir()
	executor := newToolExecutor(dir)

	// Create a file that exceeds the max size
	bigContent := strings.Repeat("x", maxFileSize+1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.txt"), []byte(bigContent), 0644))

	result := executor.readFile(map[string]interface{}{
		"path": "big.txt",
	})
	require.Contains(t, result, "file is too large")

	// A file under the limit should work fine
	smallContent := "hello world"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "small.txt"), []byte(smallContent), 0644))

	result = executor.readFile(map[string]interface{}{
		"path": "small.txt",
	})
	require.Equal(t, smallContent, result)
}

func TestListDirectory_PathTraversal(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	result := executor.listDirectory(map[string]interface{}{
		"path": "../",
	})
	require.Contains(t, result, "outside the repository")
}

func TestGrep_PathTraversal(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	result := executor.grep(map[string]interface{}{
		"pattern": "root",
		"path":    "../../etc",
	})
	require.Contains(t, result, "outside the repository")
}

func TestGrep_FlagInjection(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	// Pattern starting with - should not be interpreted as a flag
	// thanks to the -- separator
	result := executor.grep(map[string]interface{}{
		"pattern": "-rn",
		"path":    ".",
	})
	// Should not error with "invalid option", should just return no matches
	require.NotContains(t, result, "invalid option")
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
		{name: "git prefix exec flag", args: "git log --exec=malicious"},
		{name: "git prefix ext-diff", args: "git diff --ext-diff"},
		{name: "git prefix upload-pack", args: "git fetch --upload-pack=evil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.git(map[string]interface{}{
				"args": tt.args,
			})
			require.Contains(t, result, "not allowed for security reasons")
		})
	}
}

func TestGit_StripGitPrefix(t *testing.T) {
	dir := t.TempDir()
	executor := newToolExecutor(dir)

	// "git status" should be treated the same as "status"
	result := executor.git(map[string]interface{}{
		"args": "git status",
	})
	require.NotContains(t, result, "is not allowed")

	// "git push" should still be blocked
	result = executor.git(map[string]interface{}{
		"args": "git push",
	})
	require.Contains(t, result, "is not allowed")

	// bare "git" with nothing after should error
	result = executor.git(map[string]interface{}{
		"args": "git",
	})
	require.Contains(t, result, "empty git command")
}

func TestGit_BlockedSubcommand(t *testing.T) {
	dir := setupTestRepo(t)
	executor := newToolExecutor(dir)

	result := executor.git(map[string]interface{}{
		"args": "push origin main",
	})
	require.Contains(t, result, "not allowed")
}

func TestGit_AllowedSubcommands(t *testing.T) {
	for subcmd := range allowedGitSubcommands {
		t.Run(subcmd, func(t *testing.T) {
			// Just verify the subcommand is not rejected by our allowlist check.
			// The git command itself may fail (not a real repo) but the error
			// should come from git, not from our validation.
			dir := t.TempDir()
			executor := newToolExecutor(dir)

			result := executor.git(map[string]interface{}{
				"args": subcmd,
			})
			require.NotContains(t, result, "is not allowed")
		})
	}
}

package llmclient

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

const maxFileSize = 500 * 1024 // 500KB

// Git subcommands that are allowed
var allowedGitSubcommands = map[string]bool{
	"log":       true,
	"show":      true,
	"diff":      true,
	"status":    true,
	"ls-files":  true,
	"blame":     true,
	"rev-parse": true,
	"cat-file":  true,
	"checkout":  true,
	"fetch":     true,
	"pull":      true,
	"branch":    true,
	"tag":       true,
}

// Git flags that could execute arbitrary commands
var blockedGitFlags = []string{
	"--exec",
	"--ext-diff",
	"--upload-pack",
	"--receive-pack",
	"-c",
	"--config",
	"--hook",
	"--run",
}

// buildAgenticTools returns the list of tools available to the agent
func buildAgenticTools() []llms.Tool {
	return []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "read_file",
				Description: "Read the contents of a file at the given path",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "The relative path to the file to read",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "list_directory",
				Description: "List files and directories at the given path",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "The relative path to the directory to list (use '.' for root)",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "grep",
				Description: "Search for a pattern in files. Returns matching lines with file names and line numbers.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "The pattern to search for",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Optional: directory or file to search in (defaults to '.')",
						},
					},
					"required": []string{"pattern"},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "git",
				Description: "Execute a git command. Only allowed commands: log, show, diff, status, ls-files, blame, rev-parse, cat-file, checkout, fetch, pull, branch, tag.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"args": map[string]interface{}{
							"type":        "string",
							"description": "The git command arguments (e.g., 'log -n 5' or 'show HEAD')",
						},
					},
					"required": []string{"args"},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "submit_answer",
				Description: "Submit your final answer to the question. Use this when you have gathered enough information.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"answer": map[string]interface{}{
							"type":        "string",
							"description": "Your detailed answer explaining your findings",
						},
						"short_answer": map[string]interface{}{
							"type":        "boolean",
							"description": "A boolean true/false answer to yes/no questions",
						},
						"files": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "List of relevant files (optional)",
						},
						"code_snippet": map[string]interface{}{
							"type":        "string",
							"description": "A relevant code snippet (optional)",
						},
					},
					"required": []string{"answer", "short_answer"},
				},
			},
		},
	}
}

// toolExecutor handles execution of tools within a repository path
type toolExecutor struct {
	repoPath string
}

// newToolExecutor creates a new tool executor for the given repository path
func newToolExecutor(repoPath string) *toolExecutor {
	return &toolExecutor{repoPath: repoPath}
}

// execute runs a tool and returns the result as a string
func (e *toolExecutor) execute(toolName, argsJSON string) string {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("Error parsing arguments: %v", err)
	}

	switch toolName {
	case "read_file":
		return e.readFile(args)
	case "list_directory":
		return e.listDirectory(args)
	case "grep":
		return e.grep(args)
	case "git":
		return e.git(args)
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName)
	}
}

// validatePath checks that the resolved path is within the repository directory.
// Returns the validated absolute path or an error string.
func (e *toolExecutor) validatePath(path string) (string, string) {
	fullPath := filepath.Join(e.repoPath, path)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Sprintf("Error resolving path: %v", err)
	}

	absRepo, err := filepath.Abs(e.repoPath)
	if err != nil {
		return "", fmt.Sprintf("Error resolving repo path: %v", err)
	}

	// Check before resolving symlinks to catch plain traversal (e.g. "../..").
	if absPath != absRepo && !strings.HasPrefix(absPath, absRepo+string(os.PathSeparator)) {
		return "", "Error: path is outside the repository"
	}

	// Resolve symlinks and check again to prevent a symlink inside the repo
	// from pointing to a target outside the repo.
	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Sprintf("Error resolving path: %v", err)
	}
	absRepo, err = filepath.EvalSymlinks(absRepo)
	if err != nil {
		return "", fmt.Sprintf("Error resolving repo path: %v", err)
	}

	if absPath != absRepo && !strings.HasPrefix(absPath, absRepo+string(os.PathSeparator)) {
		return "", "Error: path is outside the repository"
	}

	return absPath, ""
}

func (e *toolExecutor) readFile(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok {
		return "Error: path is required"
	}

	fullPath, errMsg := e.validatePath(path)
	if errMsg != "" {
		return errMsg
	}

	debugLog("AgenticClient: read_file %s", fullPath)

	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}
	if info.Size() > maxFileSize {
		return fmt.Sprintf("Error: file is too large (%d bytes, limit is %d bytes). Try reading a smaller file or use grep to find specific content.", info.Size(), maxFileSize)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	return string(content)
}

func (e *toolExecutor) listDirectory(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok {
		path = "."
	}

	fullPath, errMsg := e.validatePath(path)
	if errMsg != "" {
		return errMsg
	}

	debugLog("AgenticClient: list_directory %s", fullPath)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return fmt.Sprintf("Error listing directory: %v", err)
	}

	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(entry.Name() + "/\n")
		} else {
			result.WriteString(entry.Name() + "\n")
		}
	}

	return result.String()
}

func (e *toolExecutor) grep(args map[string]interface{}) string {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return "Error: pattern is required"
	}

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	fullPath, errMsg := e.validatePath(path)
	if errMsg != "" {
		return errMsg
	}

	debugLog("AgenticClient: grep '%s' in %s", pattern, fullPath)

	// Use -- to prevent pattern from being interpreted as flags
	cmd := exec.Command("grep", "-rn", "--", pattern, fullPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// grep returns exit code 1 if no matches found
		if len(output) == 0 {
			return "No matches found"
		}
	}

	return string(output)
}

func (e *toolExecutor) git(args map[string]interface{}) string {
	argsStr, ok := args["args"].(string)
	if !ok || argsStr == "" {
		return "Error: git args are required"
	}

	parts := strings.Fields(argsStr)
	if len(parts) == 0 {
		return "Error: empty git command"
	}

	// Strip leading "git" if the LLM included it (e.g. "git diff" instead of "diff")
	if parts[0] == "git" {
		parts = parts[1:]
		if len(parts) == 0 {
			return "Error: empty git command"
		}
	}

	subcommand := parts[0]

	if !allowedGitSubcommands[subcommand] {
		allowed := strings.Join(slices.Sorted(maps.Keys(allowedGitSubcommands)), ", ")
		return fmt.Sprintf("Error: git subcommand '%s' is not allowed. Allowed commands: %s", subcommand, allowed)
	}

	// Check for flags that could execute arbitrary commands
	for _, arg := range parts[1:] {
		for _, blocked := range blockedGitFlags {
			if arg == blocked || strings.HasPrefix(arg, blocked+"=") {
				return fmt.Sprintf("Error: git flag '%s' is not allowed for security reasons", arg)
			}
		}
	}

	debugLog("AgenticClient: git %s", argsStr)

	cmd := exec.Command("git", append([]string{"--no-pager"}, parts...)...)
	cmd.Dir = e.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error executing git: %v\nOutput: %s", err, string(output))
	}

	return string(output)
}

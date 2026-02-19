package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Diagnostic represents a single validation issue
type Diagnostic struct {
	Severity string `json:"Severity"`
	Title    string `json:"Title"`
	Detail   string `json:"Detail"`
	Name     string `json:"Name"`
}

// Diagnostics is a map of category name to list of diagnostics
type Diagnostics map[string][]Diagnostic

var version = "dev"

// Severity constants
const (
	SeverityError     = "error"
	SeverityWarning   = "warning"
	SeverityOK        = "ok"
	SeveritySuspected = "suspected"
)

type Input struct {
	PluginPath    string `json:"pluginPath" jsonschema:"required" jsonschema_description:"The path to the plugin directory. This can be a local file path or a URL. If it's a URL, it must be a zip file."`
	SourceCodeUri string `json:"sourceCodeUri,omitempty" jsonschema_description:"The URI of the source code. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a git repository or a zip file."`
}

type Output struct {
	PluginID    string      `json:"pluginId" jsonschema_description:"The plugin ID from plugin.json."`
	Version     string      `json:"version" jsonschema_description:"The plugin version from plugin.json."`
	Diagnostics Diagnostics `json:"diagnostics" jsonschema_description:"Detailed diagnostics grouped by category (e.g., archive, manifest, security). Each category contains a list of issues with Severity (error/warning/ok/suspected), Title (brief description), Detail (detailed explanation), and Name (machine-readable identifier)."`
}

type cliOutput struct {
	ID              string      `json:"id"`
	Version         string      `json:"version"`
	PluginValidator Diagnostics `json:"plugin-validator"`
}

func isDockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func isNpxAvailable() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

func isLocalFilePath(path string) bool {
	return strings.HasPrefix(path, "/") ||
		strings.HasPrefix(path, "./") ||
		strings.HasPrefix(path, "../") ||
		strings.HasPrefix(path, "file://") ||
		(!strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://"))
}

func ValidatePlugin(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
	log.Printf("[MCP] ValidatePlugin called - pluginPath: %s, sourceCodeUri: %s", input.PluginPath, input.SourceCodeUri)

	var useDocker bool
	var method string

	if isDockerAvailable() {
		useDocker = true
		method = "docker"
		log.Printf("[MCP] Using Docker for validation")
	} else if isNpxAvailable() {
		useDocker = false
		method = "npx"
		log.Printf("[MCP] Using npx for validation")
	} else {
		return nil, Output{}, fmt.Errorf("neither docker nor npx is available. Please install Docker or Node.js")
	}

	var cmd *exec.Cmd
	var pluginArg string

	// Docker is preferred then npx as fallback
	if useDocker {
		args := []string{"run", "--pull=always", "--rm"}

		// Mount local files if needed
		if isLocalFilePath(input.PluginPath) {
			localPath := strings.TrimPrefix(input.PluginPath, "file://")
			absPath, err := filepath.Abs(localPath)
			if err != nil {
				return nil, Output{}, fmt.Errorf("failed to resolve path: %w", err)
			}
			// mounting the archive
			args = append(args, "-v", fmt.Sprintf("%s:/archive.zip:ro", absPath))
			pluginArg = "/archive.zip"
		} else {
			pluginArg = input.PluginPath
		}

		// Mount source code if provided and local
		if input.SourceCodeUri != "" {

			if isLocalFilePath(input.SourceCodeUri) {
				sourcePath := strings.TrimPrefix(input.SourceCodeUri, "file://")
				absPath, err := filepath.Abs(sourcePath)
				if err != nil {
					return nil, Output{}, fmt.Errorf("failed to resolve source code path: %w", err)
				}
				// mounting the source code
				args = append(args, "-v", fmt.Sprintf("%s:/source:ro", absPath))
				args = append(args, "grafana/plugin-validator-cli", "-jsonOutput", "-sourceCodeUri", "file:///source", pluginArg)
			} else {
				args = append(args, "grafana/plugin-validator-cli", "-jsonOutput", "-sourceCodeUri", input.SourceCodeUri, pluginArg)
			}
		} else {
			args = append(args, "grafana/plugin-validator-cli", "-jsonOutput", pluginArg)
		}

		cmd = exec.CommandContext(ctx, "docker", args...)
		log.Printf("[MCP] Executing: docker %v", args)
	} else {
		// Using npx
		args := []string{"-y", "@grafana/plugin-validator@latest", "-jsonOutput"}

		if input.SourceCodeUri != "" {
			args = append(args, "-sourceCodeUri", input.SourceCodeUri)
		}

		args = append(args, input.PluginPath)
		cmd = exec.CommandContext(ctx, "npx", args...)
		log.Printf("[MCP] Executing: npx %v", args)
	}

	// Execute the command - capture stdout and stderr separately
	var stdout, stderr []byte
	var execErr error

	stdout, execErr = cmd.Output()

	// For exit errors (non-zero exit code), we may still have valid JSON output on stdout
	// This is expected for validation failures
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			// exitErr.Stderr contains Docker pull messages or other stderr output
			stderr = exitErr.Stderr
			// stdout should already be captured above, even with non-zero exit
		} else {
			// Real error executing the command (command not found, etc.)
			return nil, Output{}, fmt.Errorf("failed to execute validator via %s: %w", method, execErr)
		}
	}

	// Parse JSON output from stdout
	log.Printf("[MCP] Command completed, stdout length: %d, stderr length: %d", len(stdout), len(stderr))

	var cliOut cliOutput
	if err := json.Unmarshal(stdout, &cliOut); err != nil {
		log.Printf("[MCP] Failed to parse JSON: %v", err)
		// If we can't parse the output, return a generic error diagnostic
		diagnostics := Diagnostics{
			"validation": []Diagnostic{
				{
					Name:     "validation-error",
					Severity: SeverityError,
					Title:    "Plugin validation failed",
					Detail:   fmt.Sprintf("Failed to parse validator output: %v\nStdout: %s\nStderr: %s", err, string(stdout), string(stderr)),
				},
			},
		}
		return nil, Output{
			PluginID:    "unknown",
			Version:     "unknown",
			Diagnostics: diagnostics,
		}, nil
	}

	return nil, Output{
		PluginID:    cliOut.ID,
		Version:     cliOut.Version,
		Diagnostics: cliOut.PluginValidator,
	}, nil
}

func run() error {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("[MCP] Starting plugin-validator MCP server v%s", version)

	server := mcp.NewServer(&mcp.Implementation{Name: "plugin-validator", Version: version}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate_plugin",
		Description: "Validates a Grafana plugin by calling the validator CLI via Docker (with --pull=always for latest) or npx. Checks metadata, security, structure, and best practices. Returns detailed errors and warnings with actionable fix suggestions.",
	}, ValidatePlugin)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

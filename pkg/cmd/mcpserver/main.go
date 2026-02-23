package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

type Input struct {
	PluginPath    string `json:"pluginPath" jsonschema:"required" jsonschema_description:"The path to the plugin directory. This can be a local file path or a URL. If it's a URL, it must be a zip file."`
	SourceCodeUri string `json:"sourceCodeUri,omitempty" jsonschema_description:"The URI of the source code. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a git repository or a zip file."`
}

type Output struct {
	PluginID    string               `json:"pluginId" jsonschema_description:"The plugin ID from plugin.json."`
	Version     string               `json:"version" jsonschema_description:"The plugin version from plugin.json."`
	Diagnostics analysis.Diagnostics `json:"diagnostics" jsonschema_description:"Detailed diagnostics grouped by category (e.g., archive, manifest, security). Each category contains a list of issues with Severity (error/warning/ok/suspected), Title (brief description), Detail (detailed explanation), and Name (machine-readable identifier)."`
}

type cliOutput struct {
	ID              string               `json:"id"`
	Version         string               `json:"version"`
	PluginValidator analysis.Diagnostics `json:"plugin-validator"`
}

func isNpxAvailable() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

func ValidatePlugin(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
	log.Printf("[MCP] ValidatePlugin called - pluginPath: %s, sourceCodeUri: %s", input.PluginPath, input.SourceCodeUri)

	if !isNpxAvailable() {
		return nil, Output{}, fmt.Errorf("npx is not available. Please install Node.js")
	}

	args := []string{"-y", "@grafana/plugin-validator@latest", "-jsonOutput"}

	if input.SourceCodeUri != "" {
		args = append(args, "-sourceCodeUri", input.SourceCodeUri)
	}

	args = append(args, input.PluginPath)
	cmd := exec.CommandContext(ctx, "npx", args...)
	log.Printf("[MCP] Executing: npx %v", args)

	// Execute the command - capture stdout and stderr separately
	var stdout, stderr []byte
	var execErr error

	stdout, execErr = cmd.Output()

	// For exit errors (non-zero exit code), we may still have valid JSON output on stdout
	// This is expected for validation failures
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			stderr = exitErr.Stderr
		} else {
			return nil, Output{}, fmt.Errorf("failed to execute validator via npx: %w", execErr)
		}
	}

	// Parse JSON output from stdout
	log.Printf("[MCP] Command completed, stdout length: %d, stderr length: %d", len(stdout), len(stderr))

	var cliOut cliOutput
	if err := json.Unmarshal(stdout, &cliOut); err != nil {
		log.Printf("[MCP] Failed to parse JSON: %v", err)
		// If we can't parse the output, return a generic error diagnostic
		diagnostics := analysis.Diagnostics{
			"validation": []analysis.Diagnostic{
				{
					Name:     "validation-error",
					Severity: analysis.Error,
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
		Description: "Validates a Grafana plugin by calling the validator CLI via npx (@grafana/plugin-validator@latest). Checks metadata, security, structure, and best practices. Returns detailed errors and warnings with actionable fix suggestions.",
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

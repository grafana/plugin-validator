package main

import (
	"context"
	"encoding/json"
	"fmt"
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

// Severity constants
const (
	SeverityError     = "error"
	SeverityWarning   = "warning"
	SeverityOK        = "ok"
	SeveritySuspected = "suspected"
)

type Input struct {
	PluginPath    string `json:"pluginPath" jsonschema:"required,description=The path to the plugin directory. This can be a local file path or a URL. If it's a URL, it must be a zip file."`
	SourceCodeUri string `json:"sourceCodeUri,omitempty" jsonschema:"description=The URI of the source code. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a git repository or a zip file."`
}

type DiagnosticSummary struct {
	TotalCategories int `json:"totalCategories" jsonschema:"description=Number of diagnostic categories checked."`
	ErrorCount      int `json:"errorCount" jsonschema:"description=Number of error-level issues found."`
	WarningCount    int `json:"warningCount" jsonschema:"description=Number of warning-level issues found."`
	OkCount         int `json:"okCount" jsonschema:"description=Number of checks that passed."`
	SuspectedCount  int `json:"suspectedCount" jsonschema:"description=Number of suspected/informational issues."`
	TotalIssues     int `json:"totalIssues" jsonschema:"description=Total number of all issues across all severity levels."`
}

type Output struct {
	PluginID    string            `json:"pluginId" jsonschema:"description=The plugin ID from plugin.json."`
	Version     string            `json:"version" jsonschema:"description=The plugin version from plugin.json."`
	Summary     DiagnosticSummary `json:"summary" jsonschema:"description=Summary statistics of the validation results."`
	Diagnostics Diagnostics       `json:"diagnostics" jsonschema:"description=Detailed diagnostics grouped by category (e.g., archive, manifest, security). Each category contains a list of issues with Severity (error/warning/ok/suspected), Title (brief description), Detail (detailed explanation), and Name (machine-readable identifier)."`
	Passed      bool              `json:"passed" jsonschema:"description=True if validation passed (no errors), false otherwise."`
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

func ValidatePlugin(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
	var useDocker bool
	var method string

	if isDockerAvailable() {
		useDocker = true
		method = "docker"
	} else if isNpxAvailable() {
		useDocker = false
		method = "npx"
	} else {
		return nil, Output{}, fmt.Errorf("neither docker nor npx is available. Please install Docker or Node.js")
	}

	var cmd *exec.Cmd
	var pluginArg string

	// Handle local file paths - need to mount for Docker
	// Check if it's a local file path (absolute, relative, or file:// URI)
	isLocalFile := strings.HasPrefix(input.PluginPath, "/") ||
		strings.HasPrefix(input.PluginPath, "./") ||
		strings.HasPrefix(input.PluginPath, "../") ||
		strings.HasPrefix(input.PluginPath, "file://") ||
		(!strings.HasPrefix(input.PluginPath, "http://") && !strings.HasPrefix(input.PluginPath, "https://"))

	// Docker is preferred then npx as fallback
	if useDocker {
		args := []string{"run", "--pull=always", "--rm"}

		// Mount local files if needed
		if isLocalFile {
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
			isLocalSource := strings.HasPrefix(input.SourceCodeUri, "/") ||
				strings.HasPrefix(input.SourceCodeUri, "./") ||
				strings.HasPrefix(input.SourceCodeUri, "../") ||
				strings.HasPrefix(input.SourceCodeUri, "file://") ||
				(!strings.HasPrefix(input.SourceCodeUri, "http://") && !strings.HasPrefix(input.SourceCodeUri, "https://"))

			if isLocalSource {
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
	} else {
		// Using npx
		args := []string{"-y", "@grafana/plugin-validator@latest", "-jsonOutput"}

		if input.SourceCodeUri != "" {
			args = append(args, "-sourceCodeUri", input.SourceCodeUri)
		}

		args = append(args, input.PluginPath)
		cmd = exec.CommandContext(ctx, "npx", args...)
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
	var cliOut cliOutput
	if err := json.Unmarshal(stdout, &cliOut); err != nil {
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
			Summary:     calculateSummary(diagnostics),
			Passed:      false,
		}, nil
	}

	// Calculate summary statistics
	summary := calculateSummary(cliOut.PluginValidator)

	return nil, Output{
		PluginID:    cliOut.ID,
		Version:     cliOut.Version,
		Summary:     summary,
		Diagnostics: cliOut.PluginValidator,
		Passed:      summary.ErrorCount == 0,
	}, nil
}

// calculateSummary computes summary statistics from diagnostics
func calculateSummary(diags Diagnostics) DiagnosticSummary {
	summary := DiagnosticSummary{
		TotalCategories: len(diags),
	}

	for _, items := range diags {
		for _, d := range items {
			switch d.Severity {
			case SeverityError:
				summary.ErrorCount++
			case SeverityWarning:
				summary.WarningCount++
			case SeverityOK:
				summary.OkCount++
			default: // "suspected" and others
				summary.SuspectedCount++
			}
		}
	}

	summary.TotalIssues = summary.ErrorCount + summary.WarningCount + summary.SuspectedCount
	return summary
}

func run() error {
	server := mcp.NewServer(&mcp.Implementation{Name: "plugin-validator", Version: "0.1.0"}, nil)
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

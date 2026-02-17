package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Input struct {
	PluginPath    string `json:"pluginPath" jsonschema:"required,description=The path to the plugin directory. This can be a local file path or a URL. If it's a URL, it must be a zip file."`
	SourceCodeUri string `json:"sourceCodeUri" jsonschema:"description=The URI of the source code. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a git repository or a zip file."`
}

type Output struct {
	Diagnostics analysis.Diagnostics `json:"diagnostics" jsonschema:"description=The diagnostics results of the plugin validation. This includes errors, warnings, and recommendations for improving the plugin."`
}

func ValidatePlugin(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
	res, err := service.ValidatePlugin(
		service.Params{
			PluginURL:     input.PluginPath,
			SourceCodeUri: input.SourceCodeUri,
		},
	)
	if err != nil {
		// Need to return diagnostics even in case of error, to provide feedback on what went wrong
		diagnostics := analysis.Diagnostics{
			"validation": []analysis.Diagnostic{
				{
					Name:     "validation-error",
					Severity: analysis.Error,
					Title:    "Plugin validation failed",
					Detail:   err.Error(),
				},
			},
		}
		return nil, Output{Diagnostics: diagnostics}, nil
	}
	return nil, Output{Diagnostics: res.Diagnostics}, nil
}

func run() error {
	server := mcp.NewServer(&mcp.Implementation{Name: "plugin-validator", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate_plugin",
		Description: "Validates a Grafana plugin against publishing requirements. Checks metadata, security, structure, and best practices. Returns detailed errors and warnings with actionable fix suggestions.",
	}, ValidatePlugin)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("failed to run: %v", err)
	}
}

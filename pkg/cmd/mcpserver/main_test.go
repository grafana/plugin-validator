package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestValidatePlugin_InvalidZip(t *testing.T) {
	archivePath := filepath.Join("..", "plugincheck2", "testdata", "invalid.zip")
	input := Input{
		PluginPath:    archivePath,
		SourceCodeUri: "",
	}
	req := &mcp.CallToolRequest{}

	_, output, err := ValidatePlugin(context.Background(), req, input)
	if err != nil {
		t.Logf("Got error (this might be expected): %v", err)
	}

	// Check that diagnostics contain error-level issues
	hasError := false
	for _, diags := range output.Diagnostics {
		for _, d := range diags {
			if d.Severity == "error" {
				hasError = true
				t.Logf("Found error diagnostic: %s - %s", d.Title, d.Detail)
			}
		}
	}

	if !hasError {
		t.Error("Expected error-level diagnostics for invalid zip, got none")
	}
}

func TestValidatePlugin_ValidZip(t *testing.T) {
	archivePath := filepath.Join("..", "plugincheck2", "testdata", "alexanderzobnin-zabbix-app-4.4.9.linux_amd64.zip")
	input := Input{
		PluginPath:    archivePath,
		SourceCodeUri: "",
	}
	req := &mcp.CallToolRequest{}

	_, output, err := ValidatePlugin(context.Background(), req, input)
	if err != nil {
		t.Fatalf("ValidatePlugin returned error: %v", err)
	}

	if len(output.Diagnostics) == 0 {
		t.Errorf("Expected diagnostics, got none")
	}
	t.Logf("Got %d diagnostic groups", len(output.Diagnostics))
}

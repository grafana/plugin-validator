# Plugin Validator MCP Server

An MCP (Model Context Protocol) server that provides Grafana plugin validation capabilities to AI assistants and code editors.

## Building

```bash
# From the project root
go build -o bin/mcpserver ./pkg/cmd/mcpserver

# Or using mage
mage build:commands
```

## Installation

### Quick Install from Release (Recommended)

**Linux/macOS:**

Run the installation script:

```bash
curl -fsSL https://raw.githubusercontent.com/grafana/plugin-validator/main/scripts/install-mcp.sh | bash
```

Or download and inspect the script first:

```bash
wget https://raw.githubusercontent.com/grafana/plugin-validator/main/scripts/install-mcp.sh
chmod +x install-mcp.sh
./install-mcp.sh
```

### Install via Go

If you have Go installed and prefer building from source:

```bash
# Clone and build
git clone https://github.com/grafana/plugin-validator.git
cd plugin-validator
go build -o ~/.local/bin/plugin-validator-mcp ./pkg/cmd/mcpserver

# Make sure ~/.local/bin is in your PATH
export PATH="$HOME/.local/bin:$PATH"
```

## Configuration

### Claude Code (CLI & VS Code Extension - Claude code chat)

**Option 1: Global Configuration**

Add to `~/.claude.json` (shared between CLI and VS Code extension):

```json
{
  "mcpServers": {
    "plugin-validator": {
      "command": "/home/YOUR_USERNAME/.local/bin/plugin-validator-mcp",
      "args": []
    }
  }
}
```

On macOS, use:

```json
{
  "mcpServers": {
    "plugin-validator": {
      "command": "/Users/YOUR_USERNAME/.local/bin/plugin-validator-mcp",
      "args": []
    }
  }
}
```

**Option 2: Project-Scoped**

Create `.mcp.json` in your project root:

```json
{
  "plugin-validator": {
    "command": "/home/YOUR_USERNAME/.local/bin/plugin-validator-mcp",
    "args": []
  }
}
```

For more details on MCP server types and configuration, see [Claude Code Plugin Documentation](https://docs.anthropic.com/en/docs/claude-code).

### VS Code Extensions

This MCP server is compatible with any VS Code extension that supports the Model Context Protocol. Below are configurations for popular extensions:

#### GitHub Copilot Chat

Check the [GitHub Copilot documentation](https://docs.github.com/en/copilot) for MCP server configuration. If GitHub Copilot supports MCP in your version, you can typically configure it via `.vscode/mcp.json` in your project:

```json
{
  "servers": {
    "plugin-validator": {
      "type": "stdio",
      "command": "/home/YOUR_USERNAME/.local/bin/plugin-validator-mcp",
      "args": []
    }
  }
}
```

**Note:** MCP support in GitHub Copilot may vary by version. Check your extension's documentation for the exact configuration format.

#### Continue

Continue supports MCP servers. Edit `~/.continue/config.json`:

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "transport": {
          "type": "stdio",
          "command": "/home/YOUR_USERNAME/.local/bin/plugin-validator-mcp"
        }
      }
    ]
  }
}
```

See [Continue MCP Documentation](https://docs.continue.dev/features/model-context-protocol) for details.

### Claude Desktop (macOS)

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "plugin-validator": {
      "command": "/Users/YOUR_USERNAME/.local/bin/plugin-validator-mcp"
    }
  }
}
```

### Claude Desktop (Linux)

Edit `~/.config/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "plugin-validator": {
      "command": "/home/YOUR_USERNAME/.local/bin/plugin-validator-mcp"
    }
  }
}
```

## Usage

Once configured, you can ask your AI assistant to validate Grafana plugins:

```

Validate this Grafana plugin: /path/to/plugin.zip

```

```

Check this plugin with source code:

- Plugin: ./my-plugin.zip
- Source: https://github.com/user/my-plugin

```

## Tool Details

### validate_plugin

Validates a Grafana plugin against publishing requirements.

**Inputs:**

- `pluginPath` (required): Path or URL to the plugin archive (.zip)
- `sourceCodeUri` (optional): Path or URL to plugin source code (zip, folder, or git repo)

**Output:**

- `diagnostics`: Structured validation results with errors, warnings, and recommendations

## Troubleshooting

### Server not found

Make sure the binary path is correct:

```bash
which plugin-validator-mcp
# or
ls -la ~/.local/bin/plugin-validator-mcp
```

### Permission denied

Make the binary executable:

```bash
chmod +x ~/.local/bin/plugin-validator-mcp
```

### Test manually

Run the server directly to check for errors:

```bash
~/.local/bin/plugin-validator-mcp
# Press Ctrl+C to exit
```

## Development

Run tests:

```bash
go test ./pkg/cmd/mcpserver -v
```

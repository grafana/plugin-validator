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

### Quick Install (Linux/macOS)

```bash
# Build and install to local bin
go build -o ~/.local/bin/plugin-validator-mcp ./pkg/cmd/mcpserver

# Make sure ~/.local/bin is in your PATH
export PATH="$HOME/.local/bin:$PATH"
```

## Configuration

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

### VS Code with Continue Extension

Edit `~/.continue/config.json` (Linux/macOS):

```json
{
  "mcpServers": [
    {
      "name": "plugin-validator",
      "command": "~/.local/bin/plugin-validator-mcp"
    }
  ]
}
```

### Cline (VS Code Extension)

Edit `~/.cline/mcp_settings.json` (Linux/macOS):

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

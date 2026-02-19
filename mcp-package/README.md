# Grafana Plugin Validator MCP Server

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that provides AI assistants with the ability to validate Grafana plugins.

## Configuration

### Claude Code (CLI & VS Code Extension)

**Using NPM (Recommended):**

Add to `~/.claude.json` (shared between CLI and VS Code extension):

```json
{
  "mcpServers": {
    "grafana-plugin-validator": {
      "command": "npx",
      "args": ["-y", "@grafana/plugin-validator-mcp@latest"]
    }
  }
}
```

**Using Docker:**

```json
{
  "mcpServers": {
    "grafana-plugin-validator": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-v",
        "/var/run/docker.sock:/var/run/docker.sock",
        "grafana/plugin-validator-mcp:latest"
      ]
    }
  }
}
```

**Project-Scoped Configuration:**

Create `.mcp.json` in your project root:

```json
{
  "grafana-plugin-validator": {
    "command": "npx",
    "args": ["-y", "@grafana/plugin-validator-mcp@latest"]
  }
}
```

For more details on MCP server types and configuration, see [Claude Code Documentation](https://docs.anthropic.com/en/docs/claude-code).

### Cline (VS Code Extension)

Add this to your Cline MCP settings:

```json
{
  "mcpServers": {
    "grafana-plugin-validator": {
      "command": "npx",
      "args": ["-y", "@grafana/plugin-validator-mcp@latest"]
    }
  }
}
```

### Codex

Edit `~/.codex/config.toml` (or create `.codex/config.toml` in your project root for project-scoped configuration):

**Using NPM (Recommended):**

```toml
[mcp_servers.grafana-plugin-validator]
command = "npx"
args = ["-y", "@grafana/plugin-validator-mcp@latest"]
```

**Using Docker:**

```toml
[mcp_servers.grafana-plugin-validator]
command = "docker"
args = [
  "run", "-i", "--rm",
  "-v", "/var/run/docker.sock:/var/run/docker.sock",
  "grafana/plugin-validator-mcp:latest"
]
```

### Other MCP Clients

For other MCP-compatible AI assistants and editors, use:

```bash
npx -y @grafana/plugin-validator-mcp@latest
```

### Claude Desktop

**macOS**: Edit `~/Library/Application Support/Claude/claude_desktop_config.json`

**Linux**: Edit `~/.config/Claude/claude_desktop_config.json`

**Using NPM (Recommended):**

```json
{
  "mcpServers": {
    "grafana-plugin-validator": {
      "command": "npx",
      "args": ["-y", "@grafana/plugin-validator-mcp@latest"]
    }
  }
}
```

**Using Docker:**

```json
{
  "mcpServers": {
    "grafana-plugin-validator": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-v",
        "/var/run/docker.sock:/var/run/docker.sock",
        "grafana/plugin-validator-mcp:latest"
      ]
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

## Available Tools

The MCP server provides the following tool:

### `validate_plugin`

Validates a Grafana plugin and returns detailed diagnostics.

**Parameters:**

- `pluginPath` (required): Path or URL to the plugin archive (zip file)
- `sourceCodeUri` (optional): Path or URL to the plugin's source code for additional checks

**Example:**

```json
{
  "pluginPath": "https://github.com/example/my-plugin/releases/download/v1.0.0/my-plugin.zip",
  "sourceCodeUri": "https://github.com/example/my-plugin"
}
```

## License

Apache-2.0 License. See the [LICENSE](https://github.com/grafana/plugin-validator/blob/main/LICENSE) file for details.

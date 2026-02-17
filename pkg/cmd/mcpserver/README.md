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

**Windows (PowerShell):**

```powershell
# Download latest release
$version = (Invoke-RestMethod "https://api.github.com/repos/grafana/plugin-validator/releases/latest").tag_name
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$url = "https://github.com/grafana/plugin-validator/releases/download/$version/plugin-validator_$($version.TrimStart('v'))_windows_$arch.zip"

# Download and extract
Invoke-WebRequest -Uri $url -OutFile "$env:TEMP\plugin-validator.zip"
Expand-Archive -Path "$env:TEMP\plugin-validator.zip" -DestinationPath "$env:TEMP\plugin-validator" -Force

# Move to user bin directory
$binDir = "$env:USERPROFILE\.local\bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
Move-Item -Path "$env:TEMP\plugin-validator\plugin-validator-mcp.exe" -Destination "$binDir\" -Force

# Add to PATH if not already present
if ($env:PATH -notlike "*$binDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$env:PATH;$binDir", "User")
}
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

### Claude Code (CLI & VS Code Extension)

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

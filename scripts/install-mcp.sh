#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Installing Grafana Plugin Validator MCP Server..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  i386|i686) ARCH="386" ;;
  *)
    echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
    exit 1
    ;;
esac

if [[ "$OS" != "linux" && "$OS" != "darwin" ]]; then
  echo -e "${RED}Error: Unsupported OS: $OS${NC}"
  echo "This script is for Linux and macOS. For Windows, see the README.md"
  exit 1
fi

# Get latest release version
echo "Fetching latest release..."
VERSION=$(curl -s https://api.github.com/repos/grafana/plugin-validator/releases/latest | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4)

if [ -z "$VERSION" ]; then
  echo -e "${RED}Error: Could not fetch latest release version${NC}"
  exit 1
fi

echo -e "${GREEN}Latest version: $VERSION${NC}"

# Download release
DOWNLOAD_URL="https://github.com/grafana/plugin-validator/releases/download/${VERSION}/plugin-validator_${VERSION#v}_${OS}_${ARCH}.tar.gz"
echo "Downloading from: $DOWNLOAD_URL"

if ! curl -fL "$DOWNLOAD_URL" -o /tmp/plugin-validator.tar.gz; then
  echo -e "${RED}Error: Failed to download release${NC}"
  exit 1
fi

# Extract MCP server binary
echo "Extracting plugin-validator-mcp binary..."
if ! tar -xzf /tmp/plugin-validator.tar.gz -C /tmp plugin-validator-mcp 2>/dev/null; then
  echo -e "${RED}Error: Failed to extract binary. The MCP server might not be included in this release.${NC}"
  echo -e "${YELLOW}Please ensure you're using version v0.38.0 or later, or build from source.${NC}"
  rm -f /tmp/plugin-validator.tar.gz
  exit 1
fi

# Install to ~/.local/bin
INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "$INSTALL_DIR"
mv /tmp/plugin-validator-mcp "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/plugin-validator-mcp"
rm /tmp/plugin-validator.tar.gz

echo -e "${GREEN}✓ Installed to $INSTALL_DIR/plugin-validator-mcp${NC}"

# Check if ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
  echo "Add the following to your ~/.bashrc or ~/.zshrc:"
  echo ""
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
  echo ""
fi

echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. Configure the MCP server in your AI assistant (see README.md)"
echo "  2. Test the installation: plugin-validator-mcp"
echo ""
echo "For configuration examples, visit:"
echo "  https://github.com/grafana/plugin-validator/blob/main/pkg/cmd/mcpserver/README.md"

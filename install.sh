#!/usr/bin/env bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

# Convert to GoReleaser naming convention
case "$OS" in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        OS="windows"
        ;;
    *)
        echo -e "${RED}Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Set version (use latest if not specified)
VERSION="${1:-latest}"

echo -e "${BLUE}Installing git-wt for $OS-$ARCH${NC}"

# Determine binary name
BINARY_NAME="git-wt"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="git-wt.exe"
fi

# Download URL
if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/shogs/git-wt/releases/latest/download/git-wt_${OS}_${ARCH}.tar.gz"
else
    DOWNLOAD_URL="https://github.com/shogs/git-wt/releases/download/${VERSION}/git-wt_${OS}_${ARCH}.tar.gz"
fi

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

echo -e "${BLUE}Downloading from $DOWNLOAD_URL${NC}"

# Download and extract
if command -v curl &> /dev/null; then
    curl -fsSL "$DOWNLOAD_URL" | tar -xz -C "$TMP_DIR"
elif command -v wget &> /dev/null; then
    wget -qO- "$DOWNLOAD_URL" | tar -xz -C "$TMP_DIR"
else
    echo -e "${RED}Error: curl or wget is required${NC}"
    exit 1
fi

# Determine installation directory
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -d "$HOME/bin" ]; then
    INSTALL_DIR="$HOME/bin"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="$HOME/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Install binary
echo -e "${BLUE}Installing to $INSTALL_DIR${NC}"
mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/git-wt"
chmod +x "$INSTALL_DIR/git-wt"

echo -e "${GREEN}✓ git-wt installed successfully!${NC}"
echo ""
echo -e "Run ${BLUE}git-wt --help${NC} to get started"
echo ""

# Check if in PATH
if ! command -v git-wt &> /dev/null; then
    echo -e "${YELLOW}⚠ Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo "Add this to your shell profile:"
    echo -e "${BLUE}export PATH=\"$INSTALL_DIR:\$PATH\"${NC}"
fi

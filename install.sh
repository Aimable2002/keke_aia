#!/bin/bash
set -e

# Keke CLI Installer for macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/Aimable2002/keke_aia/main/install.sh | bash

GITHUB_OWNER="Aimable2002"
GITHUB_REPO="keke_aia"
BINARY_NAME="keke"

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

info() {
    echo -e "${CYAN}  ► $1${NC}"
}

success() {
    echo -e "${GREEN}  ✓ $1${NC}"
}

warn() {
    echo -e "${YELLOW}  ⚠ $1${NC}"
}

error() {
    echo -e "${RED}  ✗ $1${NC}"
    exit 1
}

echo ""
echo -e "${CYAN}  ╔══════════════════════════════════╗${NC}"
echo -e "${CYAN}  ║     Keke CLI Installer v2.0      ║${NC}"
echo -e "${CYAN}  ║  AI Developer in Your Terminal   ║${NC}"
echo -e "${CYAN}  ╚══════════════════════════════════╝${NC}"
echo ""

info "Detecting system..."

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)     OS="linux";;
    Darwin*)    OS="darwin";;
    *)          error "Unsupported operating system: $OS";;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)     ARCH="amd64";;
    aarch64)    ARCH="arm64";;
    arm64)      ARCH="arm64";;
    *)          error "Unsupported architecture: $ARCH";;
esac

info "System: $OS / $ARCH"

# Determine installation directory
if [ "$OS" = "darwin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
fi

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

info "Checking latest version..."

# Get latest version from GitHub API
if command -v curl >/dev/null 2>&1; then
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_OWNER/$GITHUB_REPO/releases" | grep '"tag_name":' | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/')
elif command -v wget >/dev/null 2>&1; then
    LATEST_VERSION=$(wget -qO- "https://api.github.com/repos/$GITHUB_OWNER/$GITHUB_REPO/releases" | grep '"tag_name":' | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/')
else
    error "Neither curl nor wget found. Please install one of them."
fi

if [ -z "$LATEST_VERSION" ]; then
    error "Could not determine latest version"
fi

info "Latest version: $LATEST_VERSION"

# Construct download URL
ARCHIVE_NAME="keke_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$GITHUB_OWNER/$GITHUB_REPO/releases/download/$LATEST_VERSION/$ARCHIVE_NAME"

# Create temporary directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

info "Downloading $ARCHIVE_NAME..."

# Download with curl or wget
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/keke.tar.gz" || error "Download failed"
elif command -v wget >/dev/null 2>&1; then
    wget -q "$DOWNLOAD_URL" -O "$TMP_DIR/keke.tar.gz" || error "Download failed"
fi

success "Download complete"

info "Extracting..."
tar -xzf "$TMP_DIR/keke.tar.gz" -C "$TMP_DIR" || error "Extraction failed"

# Find the binary (it should be in the temp directory)
BINARY_PATH="$TMP_DIR/$BINARY_NAME"
if [ ! -f "$BINARY_PATH" ]; then
    error "Binary '$BINARY_NAME' not found in archive"
fi

info "Installing to $INSTALL_DIR..."
cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Add to PATH if needed
SHELL_CONFIG=""
SHELL_NAME=$(basename "$SHELL")

case "$SHELL_NAME" in
    bash)
        if [ -f "$HOME/.bashrc" ]; then
            SHELL_CONFIG="$HOME/.bashrc"
        elif [ -f "$HOME/.bash_profile" ]; then
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
        ;;
    zsh)
        SHELL_CONFIG="$HOME/.zshrc"
        ;;
    fish)
        SHELL_CONFIG="$HOME/.config/fish/config.fish"
        ;;
esac

PATH_ALREADY_SET=0
if [ -n "$SHELL_CONFIG" ] && [ -f "$SHELL_CONFIG" ]; then
    if grep -q "$INSTALL_DIR" "$SHELL_CONFIG"; then
        PATH_ALREADY_SET=1
    fi
fi

if [ $PATH_ALREADY_SET -eq 0 ]; then
    info "Adding to PATH..."
    if [ -n "$SHELL_CONFIG" ]; then
        case "$SHELL_NAME" in
            fish)
                echo "set -gx PATH \$PATH $INSTALL_DIR" >> "$SHELL_CONFIG"
                ;;
            *)
                echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
                ;;
        esac
        success "Added to PATH in $SHELL_CONFIG"
    else
        warn "Could not detect shell config. Please add $INSTALL_DIR to your PATH manually."
    fi
fi

# Also add to current session
export PATH="$PATH:$INSTALL_DIR"

echo ""
success "Keke CLI installed successfully!"
echo ""

# Try to get version
if command -v keke >/dev/null 2>&1; then
    VERSION_OUTPUT=$(keke version 2>&1 || echo "unknown")
    info "Version: $VERSION_OUTPUT"
fi

echo ""
echo -e "${GREEN}  ╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}  ║          Quick Start Guide             ║${NC}"
echo -e "${GREEN}  ╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}  1. Reload your shell:${NC}"
echo -e "${CYAN}     source $SHELL_CONFIG${NC}"
echo -e "${GRAY}     or open a new terminal window${NC}"
echo ""
echo -e "${YELLOW}  2. Navigate to your project:${NC}"
echo -e "${CYAN}     cd your-project${NC}"
echo ""
echo -e "${YELLOW}  3. Initialize Keke:${NC}"
echo -e "${CYAN}     keke init${NC}"
echo ""
echo -e "${YELLOW}  4. Login to your account:${NC}"
echo -e "${CYAN}     keke login${NC}"
echo ""
echo -e "${YELLOW}  5. Start using Keke:${NC}"
echo -e "${CYAN}     keke ask \"your question here\"${NC}"
echo -e "${CYAN}     keke research \"research topic\"${NC}"
echo -e "${CYAN}     keke credits${NC}"
echo ""
echo -e "${GRAY}  Need help? Run: keke --help${NC}"
echo ""
info "Documentation: https://github.com/$GITHUB_OWNER/$GITHUB_REPO"
echo ""
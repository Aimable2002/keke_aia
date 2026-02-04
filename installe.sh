#!/bin/sh
set -e

GITHUB_OWNER="Aimable2002"
GITHUB_REPO="keke_aia"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="keke"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
DIM='\033[2m'
NC='\033[0m'

info()    { printf "${DIM}${CYAN}►${NC} %s\n" "$1"; }
success() { printf "${GREEN}✓${NC} %s\n" "$1"; }
error()   { printf "${RED}✗${NC} %s\n" "$1"; exit 1; }

printf "\n${CYAN}  Keke CLI Installer${NC}\n"
printf "${DIM}  AI developer in your terminal${NC}\n\n"

info "Detecting system..."
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  darwin) OS_NAME="darwin" ;;
  linux)  OS_NAME="linux"  ;;
  *)      error "Unsupported OS: $OS" ;;
esac

case "$ARCH" in
  x86_64|amd64)  ARCH_NAME="amd64" ;;
  aarch64|arm64) ARCH_NAME="arm64" ;;
  *)             error "Unsupported architecture: $ARCH" ;;
esac

info "System: $OS_NAME / $ARCH_NAME"

info "Checking latest version..."
LATEST_VERSION=$(curl -sI "https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest" \
  | grep -i "location:" \
  | sed 's/.*\///' \
  | tr -d '\r\n')

if [ -z "$LATEST_VERSION" ]; then
  error "Could not determine latest version"
fi

info "Latest version: $LATEST_VERSION"

ARCHIVE_NAME="keke_${OS_NAME}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/${ARCHIVE_NAME}"
CHECKSUM_URL="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/keke_checksums.txt"

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

info "Downloading checksums..."
curl -sL "$CHECKSUM_URL" -o "$TMP_DIR/checksums.txt" || error "Failed to download checksums"

info "Downloading $ARCHIVE_NAME..."
curl -L "$DOWNLOAD_URL" -o "$TMP_DIR/keke.tar.gz" || error "Failed to download binary"

info "Verifying checksum..."
EXPECTED=$(grep "$ARCHIVE_NAME" "$TMP_DIR/checksums.txt" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
  error "Checksum not found"
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$TMP_DIR/keke.tar.gz" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$TMP_DIR/keke.tar.gz" | awk '{print $1}')
else
  error "Cannot verify checksum: sha256sum or shasum not found"
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  error "Checksum mismatch! Expected: $EXPECTED, Got: $ACTUAL"
fi

success "Checksum verified"

info "Extracting..."
cd "$TMP_DIR"
tar -xzf keke.tar.gz

if [ ! -f "$TMP_DIR/keke" ]; then
  error "Binary not found in archive"
fi

info "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
  cp "$TMP_DIR/keke" "$INSTALL_DIR/$BINARY_NAME"
  chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
  printf "${DIM}  Requesting sudo access...${NC}\n"
  sudo cp "$TMP_DIR/keke" "$INSTALL_DIR/$BINARY_NAME"
  sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

if command -v keke >/dev/null 2>&1; then
  success "Keke installed successfully"
  printf "\n"
  info "Version: $(keke version)"
  info "Next steps:"
  printf "  ${CYAN}1.${NC} ${DIM}cd your-project${NC}\n"
  printf "  ${CYAN}2.${NC} ${DIM}keke init${NC}\n"
  printf "  ${CYAN}3.${NC} ${DIM}keke login${NC}\n"
  printf "  ${CYAN}4.${NC} ${DIM}keke credits${NC}\n"
  printf "\n"
else
  error "Installation completed but 'keke' not in PATH"
fi
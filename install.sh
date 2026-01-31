#!/bin/bash
set -e

# EZDB Installer
# Usage: curl -sSL https://raw.githubusercontent.com/nhath/ezdb/main/install.sh | bash

REPO="minhnhat97kg/ezdb"
BINARY="ezdb"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        *) error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    echo "${OS}_${ARCH}"
}

# Get latest release version
get_latest_version() {
    curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install() {
    PLATFORM=$(detect_platform)
    info "Detected platform: $PLATFORM"

    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        error "Failed to get latest version"
    fi
    info "Latest version: $VERSION"

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}_${PLATFORM}"
    info "Downloading from: $DOWNLOAD_URL"

    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    curl -sL "$DOWNLOAD_URL" -o "$TMP_DIR/$BINARY"
    chmod +x "$TMP_DIR/$BINARY"

    # Verify binary
    if ! "$TMP_DIR/$BINARY" --help >/dev/null 2>&1; then
        warn "Binary verification failed, but continuing..."
    fi

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
    else
        info "Requesting sudo to install to $INSTALL_DIR"
        sudo mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
    fi

    info "Installed $BINARY to $INSTALL_DIR/$BINARY"
    info "Run 'ezdb' to start"
}

install

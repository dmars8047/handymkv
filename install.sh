#!/usr/bin/env bash
set -euo pipefail

REPO="dmars8047/handymkv"
BINARY="handymkv"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)
        echo "Unsupported OS: $OS" >&2
        echo "For Windows, download the binary from https://github.com/${REPO}/releases/latest" >&2
        exit 1
        ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

ASSET="${BINARY}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

echo "Downloading ${BINARY} (${OS}/${ARCH})..."

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

if command -v curl &>/dev/null; then
    curl -fsSL "$URL" -o "$TMP"
elif command -v wget &>/dev/null; then
    wget -qO "$TMP" "$URL"
else
    echo "Error: curl or wget is required" >&2
    exit 1
fi

chmod +x "$TMP"

# Install to /usr/local/bin, falling back to ~/.local/bin
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP" "${INSTALL_DIR}/${BINARY}"
elif command -v sudo &>/dev/null; then
    sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
    mv "$TMP" "${INSTALL_DIR}/${BINARY}"
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        echo "Note: ${INSTALL_DIR} is not in your PATH. Add the following to your shell profile:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
fi

echo "${BINARY} installed to ${INSTALL_DIR}/${BINARY}"
"${INSTALL_DIR}/${BINARY}" -v

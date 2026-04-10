#!/bin/bash
set -e

REPO="jlavera/mf-cli"
INSTALL_DIR="/usr/local/bin"
BINARY="mf"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Build auth header if token is available (for rate limits or pre-release access)
AUTH_HEADER=""
if [ -n "$GITHUB_TOKEN" ]; then
  AUTH_HEADER="Authorization: token $GITHUB_TOKEN"
fi

curl_auth() {
  if [ -n "$AUTH_HEADER" ]; then
    curl -fsSL -H "$AUTH_HEADER" "$@"
  else
    curl -fsSL "$@"
  fi
}

# Get latest release tag
echo "Fetching latest release..."
LATEST=$(curl_auth \
  -H "Accept: application/vnd.github.v3+json" \
  "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)

if [ -z "$LATEST" ]; then
  echo "Error: Could not find latest release."
  exit 1
fi

echo "Installing mf ${LATEST} (${OS}/${ARCH})..."

# Download and extract
TARBALL="${BINARY}_${OS}_${ARCH}.tar.gz"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

curl_auth \
  -H "Accept: application/octet-stream" \
  "https://github.com/${REPO}/releases/download/${LATEST}/${TARBALL}" \
  -o "${TMP_DIR}/${TARBALL}"

tar -xzf "${TMP_DIR}/${TARBALL}" -C "$TMP_DIR"

# Install
sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
sudo chmod +x "${INSTALL_DIR}/${BINARY}"

# macOS: remove quarantine and sign
if [ "$OS" = "darwin" ]; then
  sudo xattr -cr "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
  sudo codesign --force --sign - "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
fi

echo "Done! mf ${LATEST} installed to ${INSTALL_DIR}/${BINARY}"
echo "Run 'mf --help' to get started, and 'mf init' in your project to set up completions."

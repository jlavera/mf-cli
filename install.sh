#!/bin/bash
set -e

REPO="jlavera/mf-cli"
INSTALL_DIR="/usr/local/bin"
BINARY="mf"

# Require GITHUB_TOKEN for private repo access
if [ -z "$GITHUB_TOKEN" ]; then
  echo "Error: GITHUB_TOKEN is required to download from a private repo."
  echo "Usage: GITHUB_TOKEN=ghp_xxx bash <(curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh)"
  exit 1
fi

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release tag
echo "Fetching latest release..."
LATEST=$(curl -fsSL \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)

if [ -z "$LATEST" ]; then
  echo "Error: Could not find latest release. Check your GITHUB_TOKEN and repo name."
  exit 1
fi

echo "Installing mf ${LATEST} (${OS}/${ARCH})..."

# Download and extract
TARBALL="${BINARY}_${OS}_${ARCH}.tar.gz"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

curl -fsSL \
  -H "Authorization: token $GITHUB_TOKEN" \
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

# Save token for future updates
TOKEN_DIR="$HOME/.config/mf"
mkdir -p "$TOKEN_DIR"
echo "$GITHUB_TOKEN" > "$TOKEN_DIR/token"
chmod 600 "$TOKEN_DIR/token"
echo "Token saved to $TOKEN_DIR/token (used by 'mf update')"

echo "Run 'mf --help' to get started, and 'mf init' in your project to set up completions."

#!/usr/bin/env bash
set -euo pipefail

REPO="${GSSH_REPO:-mohit4bug/gssh}"
BINARY="${GSSH_BINARY:-gssh}"
INSTALL_DIR="${GSSH_INSTALL_DIR:-/usr/local/bin}"
LATEST_URL="${GSSH_LATEST_URL:-https://api.github.com/repos/${REPO}/releases/latest}"
DOWNLOAD_BASE_URL="${GSSH_DOWNLOAD_BASE_URL:-https://github.com}"

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

version="${GSSH_VERSION:-}"
if [ -z "$version" ]; then
  printf 'Fetching latest version...\n'
  version="$(curl -fsSL "$LATEST_URL" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
fi

if [ -z "$version" ]; then
  echo "could not determine release version" >&2
  exit 1
fi

archive="${BINARY}_${os}_${arch}.tar.gz"
url="${DOWNLOAD_BASE_URL}/${REPO}/releases/download/${version}/${archive}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

printf 'Installing %s %s for %s/%s...\n' "$BINARY" "$version" "$os" "$arch"
curl -fsSL "$url" -o "$tmp_dir/$archive"
tar -xzf "$tmp_dir/$archive" -C "$tmp_dir"

if [ ! -f "$tmp_dir/$BINARY" ]; then
  echo "archive did not contain $BINARY" >&2
  exit 1
fi

if mkdir -p "$INSTALL_DIR" 2>/dev/null && [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "$tmp_dir/$BINARY" "$INSTALL_DIR/$BINARY"
else
  sudo mkdir -p "$INSTALL_DIR"
  sudo install -m 0755 "$tmp_dir/$BINARY" "$INSTALL_DIR/$BINARY"
fi

printf 'Installed %s to %s/%s\n' "$BINARY" "$INSTALL_DIR" "$BINARY"

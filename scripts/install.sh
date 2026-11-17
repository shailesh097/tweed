#!/usr/bin/env bash
# Install the latest tweed release into ~/.local/bin.
#
#   curl -fsSL https://raw.githubusercontent.com/shailesh097/tweed/main/scripts/install.sh | bash
#
set -euo pipefail

REPO="shailesh097/tweed"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

case "$(uname -m)" in
  x86_64)  ARCH=amd64 ;;
  aarch64) ARCH=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

VERSION="${VERSION:-$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep -m1 '"tag_name"' | cut -d'"' -f4)}"
if [[ -z "$VERSION" ]]; then
  echo "could not determine latest release" >&2
  exit 1
fi

TARBALL="tweed_${VERSION#v}_linux_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "==> Downloading tweed $VERSION ($ARCH)"
curl -fsSL -o "$TMP/$TARBALL" "$URL/$TARBALL"
curl -fsSL -o "$TMP/checksums.txt" "$URL/checksums.txt"

echo "==> Verifying checksum"
(cd "$TMP" && grep " $TARBALL\$" checksums.txt | sha256sum -c -)

tar -xzf "$TMP/$TARBALL" -C "$TMP" tweed
install -Dm755 "$TMP/tweed" "$BIN_DIR/tweed"
echo "==> Installed $BIN_DIR/tweed"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "! $BIN_DIR is not in your PATH; add it, or run $BIN_DIR/tweed directly" ;;
esac
echo "Next: tweed doctor && tweed apply --profile workstation --dry-run"

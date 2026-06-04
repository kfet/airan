#!/bin/sh
# install.sh — install airan on any Unix by downloading a pre-built binary.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/kfet/airan/main/install.sh | sh
#   PREFIX=/usr/local ./install.sh
#
# To build from source instead, clone the repo and run `make`.
# macOS users may prefer:  brew install kfet/tap/airan

set -eu

PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="$PREFIX/bin"

die() { echo "install.sh: $*" >&2; exit 1; }

mkdir -p "$BINDIR"

# Download the pre-built binary from GitHub Releases.
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
	x86_64|amd64) ARCH="amd64" ;;
	arm64|aarch64) ARCH="arm64" ;;
	i386|i686) ARCH="386" ;;
	armv7l|armv6l) ARCH="arm" ;;
	*) die "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
	darwin|linux|freebsd) ;;
	*) die "Unsupported OS: $OS" ;;
esac

# Fetch latest release tag
echo "fetching latest release version..."
TAG=$(curl -sfI "https://github.com/kfet/airan/releases/latest" | grep -i '^location:' | tr -d '\r' | awk -F/ '{print $NF}')
if [ -z "$TAG" ]; then
	# Fallback if location header parsing fails
	TAG=$(curl -s "https://api.github.com/repos/kfet/airan/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi
[ -z "$TAG" ] && die "Could not determine latest release version"

URL="https://github.com/kfet/airan/releases/download/${TAG}/airan-${OS}-${ARCH}"
echo "downloading airan ${TAG} for ${OS}/${ARCH}..."
curl -fLo "$BINDIR/airan" "$URL" || die "Failed to download binary from $URL"
chmod +x "$BINDIR/airan"

echo "installed: $BINDIR/airan"
case ":$PATH:" in
	*":$BINDIR:"*) ;;
	*) echo "note: $BINDIR is not on your \$PATH — add it to use 'airan' directly." ;;
esac
case ":$PATH:" in
	*":$BINDIR:"*) ;;
	*) echo "note: $BINDIR is not on your \$PATH — add it to use 'airan' directly." ;;
esac

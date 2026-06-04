#!/bin/sh
# install.sh — install airan on any Unix.
#
# Usage:
#   ./install.sh                 # from a clone (builds from source, requires Go)
#   PREFIX=/usr/local ./install.sh
#   curl -fsSL https://raw.githubusercontent.com/kfet/airan/main/install.sh | sh   # standalone (downloads pre-built binary)
#
# macOS users may prefer:  brew install kfet/tap/airan

set -eu

REPO="github.com/kfet/airan"
PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="$PREFIX/bin"

die() { echo "install.sh: $*" >&2; exit 1; }

mkdir -p "$BINDIR"

if [ -f "go.mod" ] && grep -q "module $REPO" go.mod 2>/dev/null; then
	# Running from a clone: build from the working tree.
	command -v go >/dev/null 2>&1 || die "Go toolchain not found — install from https://go.dev/dl/ to build from source"
	echo "building airan from source tree…"
	go build -trimpath -o "$BINDIR/airan" ./cmd/airan
else
	# Standalone: download the pre-built binary from GitHub Releases.
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
fi

echo "installed: $BINDIR/airan"
case ":$PATH:" in
	*":$BINDIR:"*) ;;
	*) echo "note: $BINDIR is not on your \$PATH — add it to use 'airan' directly." ;;
esac

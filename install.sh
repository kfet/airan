#!/bin/sh
# install.sh — build and install airan on any Unix.
#
# Usage:
#   ./install.sh                 # from a clone
#   PREFIX=/usr/local ./install.sh
#   curl -fsSL <raw-url>/install.sh | sh   # standalone (clones first)
#
# macOS users may prefer:  brew install kfet/tap/airan
#
# Requires the Go toolchain (https://go.dev/dl/). No other dependencies.

set -eu

REPO="github.com/kfet/airan"
PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="$PREFIX/bin"

die() { echo "install.sh: $*" >&2; exit 1; }

command -v go >/dev/null 2>&1 || die "Go toolchain not found — install from https://go.dev/dl/"

mkdir -p "$BINDIR"

if [ -f "go.mod" ] && grep -q "module $REPO" go.mod 2>/dev/null; then
	# Running from a clone: build from the working tree.
	echo "building airan from source tree…"
	go build -trimpath -o "$BINDIR/airan" ./cmd/airan
else
	# Standalone: let the Go module proxy fetch and build the latest tag.
	echo "installing airan via go install…"
	GOBIN="$BINDIR" go install "$REPO/cmd/airan@latest"
fi

echo "installed: $BINDIR/airan"
case ":$PATH:" in
	*":$BINDIR:"*) ;;
	*) echo "note: $BINDIR is not on your \$PATH — add it to use 'airan' directly." ;;
esac

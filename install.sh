#!/bin/sh
set -eu

prefix="${PREFIX:-$HOME/.local}"
bindir="$prefix/bin"

mkdir -p "$bindir"
cp "$(dirname "$0")/dist/nvim-snap" "$bindir/nvim-snap"
chmod +x "$bindir/nvim-snap"
echo "installed: $bindir/nvim-snap"

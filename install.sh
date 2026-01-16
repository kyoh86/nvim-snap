#!/bin/sh
set -eu

prefix="${PREFIX:-$HOME/.local}"
bindir="$prefix/bin"
root_dir=$(cd "$(dirname "$0")" && pwd)
bundle="$root_dir/dist/nvim-snap"

mkdir -p "$bindir"
if [ ! -f "$bundle" ]; then
  "$root_dir/scripts/bundle.sh"
fi

cp "$bundle" "$bindir/nvim-snap"
chmod +x "$bindir/nvim-snap"
echo "installed: $bindir/nvim-snap"

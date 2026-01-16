#!/bin/sh
set -eu

root_dir=$(cd "$(dirname "$0")/.." && pwd)
dist_dir="$root_dir/dist"
out_file="$dist_dir/nvim-snap"
tmp_file="$(mktemp)"

mkdir -p "$dist_dir"

luabundler bundle "$root_dir/snap.lua" \
  -p "$root_dir/?.lua" \
  -p "$root_dir/?/init.lua" \
  -o "$tmp_file"

printf '%s\n' '#!/usr/bin/env -S nvim --headless -u NONE -i NONE -l' > "$out_file"
cat "$tmp_file" >> "$out_file"
chmod +x "$out_file"
rm -f "$tmp_file"

echo "built: $out_file"

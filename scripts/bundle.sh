#!/bin/sh
set -eu

root_dir=$(cd "$(dirname "$0")/.." && pwd)
dist_dir="$root_dir/dist"
out_file="$dist_dir/nvim-snap"

mkdir -p "$dist_dir"

luabundler bundle "$root_dir/snap.lua" \
  -p "$root_dir/?.lua" \
  -p "$root_dir/?/init.lua" \
  -o "$out_file"

echo "built: $out_file"

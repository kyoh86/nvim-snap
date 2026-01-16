#!/bin/sh
set -eu

root_dir=$(cd "$(dirname "$0")/.." && pwd)
dist_dir="$root_dir/dist"

mkdir -p "$dist_dir"

luabundler bundle "$root_dir/snap.lua" \
  -p "$root_dir/?.lua" \
  -p "$root_dir/?/init.lua" \
  -o "$dist_dir/snap.lua"

echo "built: $dist_dir/snap.lua"

#!/bin/sh
set -eu

prefix="${PREFIX:-$HOME/.local}"
bindir="$prefix/bin"
sharedir="$prefix/share/nvim-snap"

mkdir -p "$bindir" "$sharedir"
cp "$(dirname "$0")/snap.lua" "$sharedir/snap.lua"

cat > "$bindir/snap" <<'EOF'
#!/bin/sh
set -eu

NVIM_BIN="${NVIM:-nvim}"
snap_lua="${NVIM_SNAP_LUA:-$HOME/.local/share/nvim-snap/snap.lua}"

exec "$NVIM_BIN" --headless -u NONE -i NONE -l "$snap_lua" "$@"
EOF

chmod +x "$bindir/snap"
echo "installed: $bindir/snap"

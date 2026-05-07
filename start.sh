#!/usr/bin/env sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

echo "GridLens Phase 1 bootstrap"
echo
echo "This script performs read-only discovery only. It does not install packages,"
echo "create WireGuard interfaces, start services, or modify system configuration."
echo

if command -v gridlens >/dev/null 2>&1; then
  exec gridlens setup wireguard --dry-run
fi

if command -v go >/dev/null 2>&1; then
  cd "$repo_dir"
  exec go run ./cmd/gridlens setup wireguard --dry-run
fi

echo "Go is required to run GridLens from source."
echo "Install Go, then run:"
echo "  go run ./cmd/gridlens setup wireguard --dry-run"

#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
GO="${ROOT}/scripts/go_with_project_toolchain.sh"

if [[ ! -x "$GO" ]]; then
  echo "missing executable: $GO" >&2
  echo "run: chmod +x scripts/go_with_project_toolchain.sh" >&2
  exit 1
fi

cd "$ROOT"

echo "==> Go toolchain"
"$GO" version

echo "==> go build ./..."
"$GO" build ./...

echo "==> go test ./... -count=1"
"$GO" test ./... -count=1

echo "==> go vet ./..."
"$GO" vet ./...

echo "All Go quality gates passed."

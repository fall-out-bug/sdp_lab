#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
GO="${ROOT}/scripts/go_with_project_toolchain.sh"
cd "$ROOT"

if [[ ! -x "$GO" ]]; then
  echo "missing executable: $GO" >&2
  echo "run: chmod +x scripts/go_with_project_toolchain.sh" >&2
  exit 1
fi

echo "==> Repository consistency fallback checks"
python3 scripts/check_repo_consistency.py --json

echo "==> Running sdp-protocol-check (strict)"
"$GO" run ./cmd/sdp-protocol-check --format json --strict

echo "==> Running sdp-doc-sync (strict)"
"$GO" run ./cmd/sdp-doc-sync --mode check --strict

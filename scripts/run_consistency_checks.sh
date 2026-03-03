#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

echo "==> Repository consistency fallback checks"
python3 scripts/check_repo_consistency.py --json

if command -v sdp-protocol-check >/dev/null 2>&1; then
  echo "==> Running sdp-protocol-check"
  sdp-protocol-check --format json
else
  echo "==> sdp-protocol-check not found in PATH (using Python fallback only)"
fi

if command -v sdp-doc-sync >/dev/null 2>&1; then
  echo "==> Running sdp-doc-sync"
  sdp-doc-sync --mode check --strict
else
  echo "==> sdp-doc-sync not found in PATH (using Python fallback only)"
fi

#!/usr/bin/env bash
# Regression test: orchestrate_k8s_issue.sh validates ISSUE format (injection fix)
# sdp_dev-j2b.1.6
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="${ROOT}/scripts/orchestrate_k8s_issue.sh"

# Invalid: semicolon (command injection)
if "${SCRIPT}" --host user@host --issue 'sdp_dev;rm -rf /' 2>/dev/null; then
  echo "FAIL: invalid ISSUE with semicolon should exit with error"
  exit 1
fi

# Invalid: space
if "${SCRIPT}" --host user@host --issue 'sdp_dev 4pg' 2>/dev/null; then
  echo "FAIL: invalid ISSUE with space should exit with error"
  exit 1
fi

# Invalid: dollar
if "${SCRIPT}" --host user@host --issue 'sdp_dev$' 2>/dev/null; then
  echo "FAIL: invalid ISSUE with $ should exit with error"
  exit 1
fi

# Valid: alphanumeric, hyphens, dots
# (will fail at ssh, but must pass validation)
out=$("${SCRIPT}" --host user@host --issue 'sdp_dev-4pg' 2>&1) || true
if [[ "${out}" == *"Error: --issue must be a valid beads ID"* ]]; then
  echo "FAIL: valid ISSUE sdp_dev-4pg should pass validation"
  exit 1
fi

echo "PASS: orchestrate_k8s_issue.sh ISSUE validation"
exit 0

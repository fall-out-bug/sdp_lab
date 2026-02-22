#!/usr/bin/env bash
# Test extract_issue_id and extract_pr_url logic for feature_to_pr.sh (sdp_dev-sod probe)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

extract_issue_id() {
  local payload="$1"
  PAYLOAD="${payload}" python3 - <<'PY'
import json
import os
import sys

payload = os.environ.get("PAYLOAD", "")
start = -1
for i, ch in enumerate(payload):
    if ch in "[{":
        start = i
        break
if start == -1:
    sys.exit(1)
obj = json.loads(payload[start:])
if isinstance(obj, list):
    obj = obj[0] if obj else {}
issue_id = obj.get("id", "")
if not issue_id:
    sys.exit(1)
print(issue_id)
PY
}

extract_pr_url() {
  local payload="$1"
  PAYLOAD="${payload}" python3 - <<'PY'
import json
import os
import re

payload = os.environ.get("PAYLOAD", "")
start = -1
for i, ch in enumerate(payload):
    if ch in "[{":
        start = i
        break
if start == -1:
    print("")
    exit(0)
obj = json.loads(payload[start:])
if isinstance(obj, list):
    obj = obj[0] if obj else {}
notes = obj.get("notes", "")
match = re.search(r"https://github.com/[^\s]+/pull/\d+", notes)
print(match.group(0) if match else "")
PY
}

echo "[test] extract_issue_id from object"
got=$(extract_issue_id 'noise{"id":"sdp_dev-abc","title":"T"}')
[[ "$got" == "sdp_dev-abc" ]] || { echo "FAIL: got $got"; exit 1; }

echo "[test] extract_issue_id from array"
got=$(extract_issue_id '[{"id":"sdp_dev-xyz"}]')
[[ "$got" == "sdp_dev-xyz" ]] || { echo "FAIL: got $got"; exit 1; }

echo "[test] extract_pr_url with PR in notes"
got=$(extract_pr_url '{"notes":"merged https://github.com/foo/bar/pull/123 done"}')
[[ "$got" == "https://github.com/foo/bar/pull/123" ]] || { echo "FAIL: got $got"; exit 1; }

echo "[test] extract_pr_url with no PR"
got=$(extract_pr_url '{"notes":"no pr here"}')
[[ -z "$got" ]] || { echo "FAIL: expected empty, got $got"; exit 1; }

echo "PASS: feature_to_pr extract logic"

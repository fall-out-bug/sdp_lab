#!/usr/bin/env bash
set -euo pipefail

JSON_OUT="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --json)
      JSON_OUT="true"
      shift
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 [--json]"
      exit 2
      ;;
  esac
done

TMP_JSON="$(mktemp)"
trap 'rm -f "${TMP_JSON}"' EXIT

{
  bd list --status open --json
  bd list --status in_progress --json
  bd list --status blocked --json
  bd list --status closed --json
} > "${TMP_JSON}"

python3 - "$JSON_OUT" "${TMP_JSON}" <<'PY'
import json
import sys

json_out = sys.argv[1].lower() == "true"
json_path = sys.argv[2]
with open(json_path, "r", encoding="utf-8") as f:
    raw = f.read().strip()

decoder = json.JSONDecoder()
idx = 0
items = []
seen = set()
while idx < len(raw):
    while idx < len(raw) and raw[idx].isspace():
        idx += 1
    if idx >= len(raw):
        break
    obj, end = decoder.raw_decode(raw, idx)
    idx = end
    if isinstance(obj, list):
        for it in obj:
            iid = it.get("id")
            if iid and iid not in seen:
                seen.add(iid)
                items.append(it)

def is_autonomy_task(it):
    return it.get("issue_type") == "task" and "autonomy" in (it.get("labels") or [])

def notes(it):
    return (it.get("notes") or "").lower()

tasks = [it for it in items if is_autonomy_task(it)]
closed = [it for it in tasks if it.get("status") == "closed"]
open_tasks = [it for it in tasks if it.get("status") == "open"]
in_progress = [it for it in tasks if it.get("status") == "in_progress"]

auto_claim = [it for it in closed if "autonomy-worker(go): claimed" in notes(it)]
with_pr = [it for it in closed if "pr created:" in notes(it)]
intervention_free = [it for it in closed if "autonomy-worker(go): claimed" in notes(it) and "pr created:" in notes(it)]

def pct(part, total):
    if total == 0:
        return 0.0
    return round((part / total) * 100.0, 1)

result = {
    "autonomy_tasks_total": len(tasks),
    "autonomy_tasks_closed": len(closed),
    "autonomy_tasks_open": len(open_tasks),
    "autonomy_tasks_in_progress": len(in_progress),
    "closed_with_auto_claim": len(auto_claim),
    "closed_with_pr_note": len(with_pr),
    "closed_intervention_free": len(intervention_free),
    "intervention_free_rate_closed_pct": pct(len(intervention_free), len(closed)),
}

if json_out:
    print(json.dumps(result, indent=2))
    sys.exit(0)

print("Autonomy SLO Report")
print(f"- total autonomy tasks: {result['autonomy_tasks_total']}")
print(f"- closed autonomy tasks: {result['autonomy_tasks_closed']}")
print(f"- open autonomy tasks: {result['autonomy_tasks_open']}")
print(f"- in_progress autonomy tasks: {result['autonomy_tasks_in_progress']}")
print(f"- closed with auto claim evidence: {result['closed_with_auto_claim']}")
print(f"- closed with PR note: {result['closed_with_pr_note']}")
print(f"- intervention-free closed tasks: {result['closed_intervention_free']}")
print(f"- intervention-free rate on closed tasks: {result['intervention_free_rate_closed_pct']}%")
PY

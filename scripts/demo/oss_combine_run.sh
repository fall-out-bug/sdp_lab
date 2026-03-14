#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
GO_TOOL="$ROOT/scripts/go_with_project_toolchain.sh"
DEMO_ROOT="$ROOT/examples/oss-combine-demo"
ARTIFACT_ROOT="$DEMO_ROOT/artifacts"
MODE="dry-run"
MAX_RUNTIME_SECONDS=1800

usage() {
  cat <<'USAGE'
Usage: scripts/demo/oss_combine_run.sh [--execute]

Options:
  --execute   Run the full demo flow against local Beads state.

Default mode is dry-run, which prints the same flow without mutating state.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --execute)
      MODE="execute"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

START_TS="$(date +%s)"
STAMP="$(date -u +%Y%m%d-%H%M%SZ)"
ARTIFACT_DIR="$ARTIFACT_ROOT/$STAMP"

run_step() {
  local label="$1"
  shift
  echo "[$label] $*"
  if [[ "$MODE" == "execute" ]]; then
    "$@"
  fi
}

check_runtime() {
  local now elapsed
  now="$(date +%s)"
  elapsed=$((now - START_TS))
  if (( elapsed > MAX_RUNTIME_SECONDS )); then
    echo "[runtime] exceeded 30 minute cap (${elapsed}s)" >&2
    exit 1
  fi
  echo "[runtime] ${elapsed}s elapsed"
}

mkdir -p "$DEMO_ROOT" "$ARTIFACT_ROOT"
if [[ "$MODE" == "execute" ]]; then
  mkdir -p "$ARTIFACT_DIR"
fi

echo "[mode] $MODE"
echo "[goal] reproducible issue-to-evidence demo with guard denial+recovery"

if [[ "$MODE" == "execute" ]]; then
  echo "[bootstrap] $GO_TOOL run ./cmd/sdp-up --profile oss-combine --dry-run"
  if ! "$GO_TOOL" run ./cmd/sdp-up --profile oss-combine --dry-run; then
    echo "[bootstrap] warning: profile dry-run failed; continuing demo flow" >&2
  fi
else
  run_step "bootstrap" "$GO_TOOL" run ./cmd/sdp-up --profile oss-combine --dry-run
fi
check_runtime

TASK_TITLE="F069 demo flow $STAMP"
TASK_DESC="Demo issue-to-lifecycle flow for OSS combine profile"
TASK_ID="sdplab-demo-$STAMP"

if [[ "$MODE" == "execute" ]]; then
  TASK_ID="$(bd create "$TASK_TITLE" -t task -p 2 --labels F069,demo,oss-combine --description "$TASK_DESC" --silent)"
  echo "[task] created $TASK_ID"
  bd update "$TASK_ID" --status in_progress >/dev/null
  echo "[task] moved to in_progress"
else
  echo "[task] would create: $TASK_TITLE"
  echo "[task] would move to in_progress"
fi

if [[ "$MODE" == "execute" ]]; then
  echo "[policy] running expected guard denial"
  if "$GO_TOOL" run ./cmd/sdp-guard --check-constraints --phase build --command "git push --force"; then
    echo "[policy] expected denial but command was allowed" >&2
    exit 1
  fi
  echo "[policy] denial captured; running recovery-safe command"
  "$GO_TOOL" run ./cmd/sdp-guard --check-constraints --phase build --command "git status"
else
  echo "[policy] would run denial command: git push --force"
  echo "[policy] would run recovery command: git status"
fi
check_runtime

if [[ "$MODE" == "execute" ]]; then
  ATTEST_PATH="$ARTIFACT_DIR/demo-auto-attest.json"
  REPORT_PATH="$ARTIFACT_DIR/demo-auto-attest-report.json"
  SUMMARY_PATH="$ARTIFACT_DIR/summary.md"

  "$GO_TOOL" run ./internal/evidence/cmd/auto-attest --base-branch master --output "$ATTEST_PATH" --report "$REPORT_PATH"

  python3 - "$TASK_ID" "$ATTEST_PATH" "$REPORT_PATH" "$SUMMARY_PATH" <<'PY'
import json
import pathlib
import sys

task_id, attest_path, report_path, summary_path = sys.argv[1:]
report = json.loads(pathlib.Path(report_path).read_text(encoding="utf-8"))
lines = [
    "# OSS Combine Demo Summary",
    "",
    f"- Task: `{task_id}`",
    f"- Attestation: `{attest_path}`",
    f"- Report: `{report_path}`",
    f"- Branch: `{report.get('branch', '')}`",
    f"- Head Commit: `{report.get('head_commit', '')}`",
    f"- Tests Pass: `{report.get('all_tests_pass', False)}`",
    f"- Lint Pass: `{report.get('all_lint_pass', False)}`",
    f"- Scope Compliance: `{report.get('scope_compliance', {}).get('ok', False)}`",
]
pathlib.Path(summary_path).write_text("\n".join(lines) + "\n", encoding="utf-8")
PY

  bd close "$TASK_ID" --reason "OSS combine demo flow completed with evidence bundle and policy denial+recovery." >/dev/null
  echo "[task] closed $TASK_ID"
  echo "[artifacts] $ARTIFACT_DIR"
else
  echo "[evidence] would run auto-attest and generate markdown summary"
  echo "[task] would close task after evidence summary"
fi

check_runtime
echo "[done] demo flow completed"

echo ""
echo "=== Failure Recovery Guidance ==="
echo ""
echo "If this demo failed, use sdp-ready for recovery instructions:"
echo ""
echo "  sdp-ready --instructions"
echo ""
echo "For machine-readable guidance:"
echo ""
echo "  sdp-ready --format status-view --instructions"
echo ""
echo "See examples/oss-combine-demo/FAILURE_RECOVERY.md for common scenarios."
echo ""

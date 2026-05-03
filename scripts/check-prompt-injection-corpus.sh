#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${SDP_PI_REPORT_DIR:-${ROOT}/.sdp/findings}"
mkdir -p "${OUT_DIR}"
REPORT="${OUT_DIR}/prompt-injection-corpus.txt"

{
  echo "F164 prompt-injection corpus check"
  echo "mode: static/mock blocking; live-provider advisory"
  echo
  echo "== static PI-013 prompt surface check =="
  "${ROOT}/scripts/prompt-injection-check.sh"
  echo
  echo "== CI workflow invariant check =="
  WORKFLOW="${ROOT}/.github/workflows/prompt-injection-corpus.yml"
  grep -q "scripts/check-prompt-injection-corpus.sh" "${WORKFLOW}"
  grep -q "cmd/sdp-pi-eval" "${WORKFLOW}"
  grep -q "upload-artifact" "${WORKFLOW}"
  ! grep -Eq "(OPENROUTER_API_KEY|ZAI_API_KEY|KIMI_API_KEY|MINIMAX_API_KEY)" "${WORKFLOW}"
  echo "PASS: workflow invokes static/mock wrapper, uploads report, and does not require live-provider secrets."
  echo
  echo "== mock/static Go regressions =="
  go test ./internal/evals
  go test -tags sdp_experimental ./cmd/sdp-eval
  go test -tags sdp_experimental ./cmd/sdp-pi-eval
  echo
  echo "== sdp-eval static/advisory report =="
  go run -tags sdp_experimental ./cmd/sdp-eval --prompt-injection-report --project-root "${ROOT}"
  echo
  echo "== live-provider eval =="
  if [ "${SDP_PI_LIVE_EVAL:-0}" = "1" ]; then
    echo "ADVISORY: live-provider eval requested. Run cmd/sdp-pi-eval manually/scheduled and attach artifacts; this wrapper does not make live availability a PR gate."
  else
    echo "ADVISORY_DEGRADED: skipped; no live-provider credentials required for CI."
  fi
} | tee "${REPORT}"

echo "report: ${REPORT}"

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
  echo "== mock/static Go regressions =="
  go test ./internal/evals
  go test -tags sdp_experimental ./cmd/sdp-eval
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

#!/usr/bin/env bash
# sdp-dispatch.sh — harness-agnostic subagent + codex dispatcher.
#
# Called by the delivery-loop skill (and other skills) to invoke subagents
# or run a codex review without the caller knowing which harness is active.
#
# Usage:
#   sdp-dispatch.sh subagent  <skill_name> [prompt...]
#   sdp-dispatch.sh codex_review <prompt...>
#
# Harness detection:
#   SDP_HARNESS env var (preferred)
#   fallback: detect via running CLI / env markers
#
# This is a thin case-branch abstraction. Drift is contained to this file —
# adding a harness = one new case in one place.

set -euo pipefail

detect_harness() {
  if [[ -n "${SDP_HARNESS:-}" ]]; then
    echo "$SDP_HARNESS"
    return
  fi
  # fallback heuristics
  if [[ -n "${CLAUDECODE:-}" || -n "${CLAUDE_CODE:-}" ]]; then echo claude-code; return; fi
  if [[ -n "${CODEX_CLI:-}" ]]; then echo codex; return; fi
  if [[ -n "${OPENCODE:-}" ]]; then echo opencode; return; fi
  if [[ -n "${CURSOR_AGENT:-}" ]]; then echo cursor; return; fi
  echo "claude-code"  # safe default; delivery-loop is most mature on claude
}

dispatch_subagent() {
  local harness="$1"; shift
  local skill="$1"; shift
  local prompt="$*"
  case "$harness" in
    claude-code)
      # In Claude Code, subagents are invoked via @<skill> inline in the main session.
      # This script emits the canonical invocation for the caller to paste.
      echo "@${skill} ${prompt}"
      ;;
    codex)
      echo "codex exec --skill ${skill} \"${prompt}\""
      ;;
    opencode)
      echo "opencode --agent ${skill} \"${prompt}\""
      ;;
    cursor)
      echo "cursor agent --skill ${skill} \"${prompt}\""
      ;;
    *)
      echo "sdp-dispatch: unknown harness: $harness" >&2
      exit 2
      ;;
  esac
}

codex_review() {
  local harness="$1"; shift
  local prompt="$*"
  case "$harness" in
    claude-code)
      # /codex:rescue is a Claude Code slash command wrapping codex exec.
      echo "/codex:rescue \"${prompt}\""
      ;;
    codex|opencode|cursor)
      echo "codex exec \"${prompt}\""
      ;;
    *)
      echo "sdp-dispatch: unknown harness: $harness" >&2
      exit 2
      ;;
  esac
}

main() {
  local action="${1:-}"; shift || true
  local harness
  harness="$(detect_harness)"

  case "$action" in
    subagent)
      dispatch_subagent "$harness" "$@"
      ;;
    codex_review)
      codex_review "$harness" "$@"
      ;;
    detect)
      echo "$harness"
      ;;
    "")
      echo "Usage: sdp-dispatch.sh {subagent|codex_review|detect} [args...]" >&2
      exit 2
      ;;
    *)
      echo "sdp-dispatch: unknown action: $action" >&2
      exit 2
      ;;
  esac
}

main "$@"

#!/usr/bin/env bash
# install-bd-post-close-hook.sh — Brownfield-safe installer for post-bd-close sync.
#
# Creates a shell alias/function `bd` that intercepts `bd close` calls and runs
# post-bd-close-sync.sh after each successful close.
#
# Install modes:
#   --shell    Print a shell snippet to eval (default)
#   --bashrc   Append to ~/.bashrc (or $BD_HOOK_RCFILE)
#   --check    Check if hook is installed
#   --uninstall Remove hook from RC file
#
# Brownfield-safe:
#   - Does not modify bd itself
#   - Only intercepts `bd close` and `bd done` subcommands
#   - Falls back to unmodified bd for all other commands
#   - Safe to run multiple times (idempotent)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
HOOK_SCRIPT="${SCRIPT_DIR}/hooks/post-bd-close-sync.sh"
MARKER_BEGIN="# --- BEGIN BD POST-CLOSE HOOK ---"
MARKER_END="# --- END BD POST-CLOSE HOOK ---"

# The shell function that wraps bd
read -r -d '' HOOK_FUNCTION <<'FUNC' || true
_bd_with_post_close() {
  local bd_bin
  bd_bin="$(command -v bd 2>/dev/null)" || true
  if [[ -z "$bd_bin" ]]; then
    echo "bd: command not found" >&2
    return 127
  fi

  # Only intercept close/done subcommands
  local subcmd="${1:-}"
  case "$subcmd" in
    close|done)
      # Run the real bd close
      "$bd_bin" "$@"
      local rc=$?
      if [[ $rc -eq 0 ]]; then
        # Extract issue IDs from args (skip flags)
        local issue_ids=()
        local arg
        for arg in "$@"; do
          case "$arg" in
            -*) continue ;;
            close|done) continue ;;
            *)
              issue_ids+=("$arg")
              ;;
          esac
        done
        if [[ ${#issue_ids[@]} -gt 0 ]]; then
          local hook_script
          hook_script="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd)/scripts/hooks/post-bd-close-sync.sh" || true
          if [[ -z "$hook_script" || ! -f "$hook_script" ]]; then
            # Try from PROJECT_ROOT env or git root
            local root="${PROJECT_ROOT:-}"
            if [[ -z "$root" ]]; then
              root="$(git rev-parse --show-toplevel 2>/dev/null)" || true
            fi
            hook_script="${root}/scripts/hooks/post-bd-close-sync.sh"
          fi
          if [[ -f "$hook_script" ]]; then
            bash "$hook_script" "${issue_ids[@]}" 2>&1 || true
          fi
        fi
      fi
      return $rc
      ;;
    *)
      # Pass through to real bd unchanged
      "$bd_bin" "$@"
      ;;
  esac
}

# Override bd with the wrapper
alias bd='_bd_with_post_close' 2>/dev/null || true
FUNC

generate_snippet() {
  printf '%s\n%s\n%s\n' "$MARKER_BEGIN" "$HOOK_FUNCTION" "$MARKER_END"
}

cmd_shell() {
  generate_snippet
}

cmd_bashrc() {
  local rcfile="${BD_HOOK_RCFILE:-$HOME/.bashrc}"

  # Remove any previous installation (idempotent)
  if [[ -f "$rcfile" ]]; then
    local tmpfile
    tmpfile="$(mktemp)"
    awk -v begin="$MARKER_BEGIN" -v end="$MARKER_END" '
      $0 == begin { skip=1; next }
      $0 == end { skip=0; next }
      !skip { print }
    ' "$rcfile" > "$tmpfile" && mv "$tmpfile" "$rcfile"
  fi

  # Append new snippet
  {
    echo ""
    generate_snippet
  } >> "$rcfile"

  echo "Installed bd post-close hook to ${rcfile}"
  echo "Run 'source ${rcfile}' or start a new shell to activate."
}

cmd_check() {
  local rcfile="${BD_HOOK_RCFILE:-$HOME/.bashrc}"

  if [[ -f "$rcfile" ]] && grep -q "$MARKER_BEGIN" "$rcfile"; then
    echo "Hook installed in ${rcfile}"
    if [[ -f "$HOOK_SCRIPT" ]]; then
      echo "Hook script: ${HOOK_SCRIPT} (OK)"
    else
      echo "Hook script: ${HOOK_SCRIPT} (MISSING)"
    fi
    return 0
  fi

  # Also check if alias is active in current shell
  if type _bd_with_post_close >/dev/null 2>&1; then
    echo "Hook active in current shell (alias)"
    return 0
  fi

  echo "Hook not installed"
  return 1
}

cmd_uninstall() {
  local rcfile="${BD_HOOK_RCFILE:-$HOME/.bashrc}"

  if [[ -f "$rcfile" ]]; then
    local tmpfile
    tmpfile="$(mktemp)"
    awk -v begin="$MARKER_BEGIN" -v end="$MARKER_END" '
      $0 == begin { skip=1; next }
      $0 == end { skip=0; next }
      !skip { print }
    ' "$rcfile" > "$tmpfile" && mv "$tmpfile" "$rcfile"
    echo "Removed bd post-close hook from ${rcfile}"
  else
    echo "No ${rcfile} found"
  fi

  # Unalias in current shell
  unalias bd 2>/dev/null || true
  unset -f _bd_with_post_close 2>/dev/null || true
  echo "Hook deactivated in current shell"
}

usage() {
  cat <<'EOF'
Usage: scripts/install-bd-post-close-hook.sh [--shell|--bashrc|--check|--uninstall]

Modes:
  --shell      Print eval-able shell snippet (default)
  --bashrc     Append hook to ~/.bashrc (or $BD_HOOK_RCFILE)
  --check      Check if hook is currently installed
  --uninstall  Remove hook from RC file and current shell

The hook wraps `bd close` and `bd done` to auto-sync workstream files
from backlog/ to done/ after each successful close.

For manual use (no RC modification):
  eval "$(scripts/install-bd-post-close-hook.sh --shell)"
EOF
}

main() {
  local mode="${1:-}"
  case "$mode" in
    --shell|"") cmd_shell ;;
    --bashrc)   cmd_bashrc ;;
    --check)    cmd_check ;;
    --uninstall) cmd_uninstall ;;
    -h|--help)  usage ;;
    *)
      echo "Unknown option: $mode" >&2
      usage >&2
      exit 2
      ;;
  esac
}

main "$@"

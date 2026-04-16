#!/usr/bin/env bash
# install_bd_hooks.sh — Install post-`bd close` hook into .git/hooks/
#
# Adds a post-bd-close hook that invokes bd_post_close.sh after `bd close`.
# Brownfield-safe: merges with existing hooks, never overwrites without --force.
#
# Usage:
#   scripts/install_bd_hooks.sh             # install
#   scripts/install_bd_hooks.sh --uninstall  # remove
#   scripts/install_bd_hooks.sh --force      # overwrite existing hook
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOK_NAME="post-bd-close"
HOOK_SRC="$REPO_ROOT/scripts/bd_post_close.sh"

# Git hooks directory: resolve through worktree if needed
if [ -d "$REPO_ROOT/.git/hooks" ]; then
  HOOKS_DIR="$REPO_ROOT/.git/hooks"
elif git -C "$REPO_ROOT" rev-parse --git-dir >/dev/null 2>&1; then
  HOOKS_DIR="$(git -C "$REPO_ROOT" rev-parse --git-dir)/hooks"
else
  echo "ERROR: Cannot find git hooks directory" >&2
  exit 1
fi

HOOK_DST="$HOOKS_DIR/$HOOK_NAME"

# ---------------------------------------------------------------------------
# Uninstall
# ---------------------------------------------------------------------------
if [ "${1:-}" = "--uninstall" ]; then
  if [ -f "$HOOK_DST" ]; then
    # Only remove if it was installed by us (contains our marker)
    if grep -q '# SDP: bd_post_close hook' "$HOOK_DST" 2>/dev/null; then
      rm -f "$HOOK_DST"
      echo "Uninstalled: ${HOOK_NAME}"
    else
      echo "Hook ${HOOK_NAME} exists but was not installed by SDP; skipping removal"
      echo "Remove manually: rm ${HOOK_DST}"
      exit 1
    fi
  else
    echo "Hook ${HOOK_NAME} not found; nothing to uninstall"
  fi
  exit 0
fi

# ---------------------------------------------------------------------------
# Validate source script exists
# ---------------------------------------------------------------------------
if [ ! -f "$HOOK_SRC" ]; then
  echo "ERROR: Hook source not found: ${HOOK_SRC}" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------
mkdir -p "$HOOKS_DIR"

FORCE=0
if [ "${1:-}" = "--force" ]; then
  FORCE=1
fi

# The hook content: a thin wrapper that pipes bd output to bd_post_close.sh
HOOK_CONTENT='# SDP: bd_post_close hook — installed by install_bd_hooks.sh
# Receives bd close output via stdin; passes bead IDs to bd_post_close.sh
BD_POST_CLOSE_SCRIPT="'"${HOOK_SRC}"'"
if [ -x "$BD_POST_CLOSE_SCRIPT" ]; then
  "$BD_POST_CLOSE_SCRIPT" "$@"
else
  echo "[bd-post-close] WARN: script not found or not executable: $BD_POST_CLOSE_SCRIPT" >&2
fi
'

if [ -f "$HOOK_DST" ] && [ "$FORCE" -ne 1 ]; then
  # Check if we already installed
  if grep -q '# SDP: bd_post_close hook' "$HOOK_DST" 2>/dev/null; then
    echo "Hook ${HOOK_NAME} already installed (up to date)"
    exit 0
  fi

  # Merge approach: append our hook content to the existing file
  echo "" >> "$HOOK_DST"
  echo "# --- SDP bd_post_close (appended by install_bd_hooks.sh) ---" >> "$HOOK_DST"
  echo "$HOOK_CONTENT" >> "$HOOK_DST"
  chmod +x "$HOOK_DST"
  echo "Merged: appended bd_post_close hook to existing ${HOOK_NAME}"
else
  # Create new hook file
  printf '%s\n' "$HOOK_CONTENT" > "$HOOK_DST"
  chmod +x "$HOOK_DST"
  if [ "$FORCE" -eq 1 ]; then
    echo "Installed (forced): ${HOOK_NAME}"
  else
    echo "Installed: ${HOOK_NAME}"
  fi
fi

# Validate bash syntax of the installed hook
if command -v bash >/dev/null 2>&1; then
  if ! bash -n "$HOOK_DST" 2>/dev/null; then
    echo "WARNING: Installed hook ${HOOK_NAME} has bash syntax errors. Review: ${HOOK_DST}" >&2
  fi
fi

# Make the source script executable too
chmod +x "$HOOK_SRC"

echo "Hook will invoke: ${HOOK_SRC}"
echo ""
echo "To uninstall: scripts/install_bd_hooks.sh --uninstall"

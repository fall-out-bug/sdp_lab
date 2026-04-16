#!/usr/bin/env bash
# install_bd_hooks.sh — Set up bd post-close wrapper for your shell
#
# Execution model:
#   bd does not support custom hooks, so we intercept the `bd close` command
#   via a shell wrapper/alias.  After `bd close` runs, the wrapper pipes the
#   bead IDs to bd_post_close.sh which moves completed workstream files.
#
# Usage:
#   scripts/install_bd_hooks.sh                 # install alias + wrapper
#   scripts/install_bd_hooks.sh --uninstall      # remove alias from shell rc
#   scripts/install_bd_hooks.sh --force          # overwrite existing wrapper
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WRAPPER_NAME="bd_wrapper.sh"
WRAPPER_SRC="$REPO_ROOT/scripts/$WRAPPER_NAME"
POST_CLOSE_SRC="$REPO_ROOT/scripts/bd_post_close.sh"
MARKER="# SDP: bd_wrapper alias"

# ---------------------------------------------------------------------------
# Create the wrapper script that intercepts `bd close` and invokes post-close
# ---------------------------------------------------------------------------
create_wrapper() {
  cat > "$WRAPPER_SRC" <<'WRAPPER'
#!/usr/bin/env bash
# bd_wrapper.sh — Intercept `bd close` and run bd_post_close.sh afterwards
#
# This wrapper proxies ALL bd commands.  When the command is `bd close`,
# it captures the output, extracts bead IDs, and pipes them to
# bd_post_close.sh for workstream auto-sync.
#
# Execution model:
#   The user sets an alias:  alias bd='/path/to/scripts/bd_wrapper.sh'
#   This makes every `bd` invocation go through this wrapper.
#   Only `bd close` triggers the post-close logic; all other commands
#   pass through transparently.
set -uo pipefail

# Resolve the directory where this script lives
_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_POST_CLOSE="$_SCRIPT_DIR/bd_post_close.sh"

# Find the real bd binary (skip our own alias)
_BD_BIN="$(command -v bd 2>/dev/null || true)"
if [ -z "$_BD_BIN" ] || [ "$_BD_BIN" = "${BASH_SOURCE[0]}" ]; then
  # Fallback: try unaliasing in a subshell and searching again
  _BD_BIN="$(unalias bd 2>/dev/null; command -v bd 2>/dev/null || true)"
fi

if [ -z "$_BD_BIN" ]; then
  echo "[bd-wrapper] ERROR: cannot find 'bd' binary in PATH" >&2
  exit 127
fi

# If the command is NOT `bd close`, just pass through
if [ "${1:-}" != "close" ]; then
  exec "$_BD_BIN" "$@"
fi

# --- bd close handling ---
# Run the real `bd close` and capture output
_output=$("$_BD_BIN" "$@" 2>&1) || true
echo "$_output"

# Extract bead IDs from the output and pipe to bd_post_close.sh
if [ -x "$_POST_CLOSE" ]; then
  echo "$_output" | REPO_ROOT="${REPO_ROOT:-}" "$_POST_CLOSE" || true
else
  echo "[bd-wrapper] WARN: bd_post_close.sh not found or not executable: $_POST_CLOSE" >&2
fi
WRAPPER

  chmod +x "$WRAPPER_SRC"
  chmod +x "$POST_CLOSE_SRC"
}

# ---------------------------------------------------------------------------
# Detect the user's shell rc file
# ---------------------------------------------------------------------------
detect_shell_rc() {
  local shell_name
  shell_name="$(basename "${SHELL:-/bin/bash}")"
  case "$shell_name" in
    zsh)  echo "${HOME}/.zshrc" ;;
    bash) echo "${HOME}/.bashrc" ;;
    *)    echo "${HOME}/.profile" ;;
  esac
}

# ---------------------------------------------------------------------------
# Uninstall
# ---------------------------------------------------------------------------
if [ "${1:-}" = "--uninstall" ]; then
  rc_file="$(detect_shell_rc)"
  if [ -f "$rc_file" ] && grep -q "$MARKER" "$rc_file" 2>/dev/null; then
    # Remove the marker line and the alias line that follows it
    sed -i.bak "/${MARKER}/d" "$rc_file"
    sed -i.bak "/alias bd=.*bd_wrapper/d" "$rc_file"
    rm -f "${rc_file}.bak"
    echo "Uninstalled: removed bd alias from ${rc_file}"
  else
    echo "No SDP bd alias found in ${rc_file}; nothing to uninstall"
  fi
  echo "Wrapper script left at: ${WRAPPER_SRC} (remove manually if desired)"
  exit 0
fi

# ---------------------------------------------------------------------------
# Validate source script exists
# ---------------------------------------------------------------------------
if [ ! -f "$POST_CLOSE_SRC" ]; then
  echo "ERROR: bd_post_close.sh not found: ${POST_CLOSE_SRC}" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------
FORCE=0
if [ "${1:-}" = "--force" ]; then
  FORCE=1
fi

# Create/update the wrapper script
if [ -f "$WRAPPER_SRC" ] && [ "$FORCE" -ne 1 ]; then
  if grep -q '# SDP: bd_wrapper.sh' "$WRAPPER_SRC" 2>/dev/null; then
    echo "Wrapper already up to date: ${WRAPPER_SRC}"
  else
    echo "WARNING: ${WRAPPER_SRC} exists but was not created by SDP. Use --force to overwrite." >&2
    exit 1
  fi
else
  create_wrapper
  if [ "$FORCE" -eq 1 ]; then
    echo "Created wrapper (forced): ${WRAPPER_SRC}"
  else
    echo "Created wrapper: ${WRAPPER_SRC}"
  fi
fi

# Validate bash syntax of the wrapper
if command -v bash >/dev/null 2>&1; then
  if ! bash -n "$WRAPPER_SRC" 2>/dev/null; then
    echo "WARNING: Wrapper has bash syntax errors. Review: ${WRAPPER_SRC}" >&2
  fi
fi

# Add alias to shell rc
rc_file="$(detect_shell_rc)"
ALIAS_LINE="alias bd='${WRAPPER_SRC}'"

if [ -f "$rc_file" ] && grep -qF "$ALIAS_LINE" "$rc_file" 2>/dev/null; then
  echo "Alias already present in ${rc_file}"
elif [ -f "$rc_file" ] && grep -q "$MARKER" "$rc_file" 2>/dev/null; then
  echo "SDP marker found in ${rc_file} but alias differs. Updating."
  # Replace the existing alias line
  sed -i.bak "s|alias bd=.*|${ALIAS_LINE}|" "$rc_file"
  rm -f "${rc_file}.bak"
  echo "Updated alias in ${rc_file}"
else
  echo "" >> "$rc_file"
  echo "$MARKER" >> "$rc_file"
  echo "$ALIAS_LINE" >> "$rc_file"
  echo "Added alias to ${rc_file}"
fi

echo ""
echo "To activate: source ${rc_file}  (or restart your shell)"
echo "To uninstall: scripts/install_bd_hooks.sh --uninstall"

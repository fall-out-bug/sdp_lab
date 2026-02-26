#!/bin/sh
# Install Git hooks: symlink .git/hooks/* to sdp/hooks/ or scripts/hooks/*.sh
# Run from repo root. Idempotent. Prefers sdp/hooks/ when sdp submodule present.
set -e

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

# Prefer sdp/hooks/ when sdp submodule has hooks (protocol source)
if [ -f "$ROOT/sdp/hooks/install-git-hooks.sh" ]; then
  exec sh "$ROOT/sdp/hooks/install-git-hooks.sh"
fi

# Fallback: scripts/hooks/ (sdp_dev without sdp or legacy)
HOOKS_DIR=".git/hooks"
SCRIPTS_DIR="scripts/hooks"
mkdir -p "$HOOKS_DIR"

if [ -f "$SCRIPTS_DIR/pre-commit.sh" ]; then
  ln -sf ../../scripts/hooks/pre-commit.sh "$HOOKS_DIR/pre-commit"
  chmod +x "$SCRIPTS_DIR/pre-commit.sh"
  echo "Installed pre-commit"
fi

if [ -f "$SCRIPTS_DIR/pre-push.sh" ]; then
  ln -sf ../../scripts/hooks/pre-push.sh "$HOOKS_DIR/pre-push"
  chmod +x "$SCRIPTS_DIR/pre-push.sh"
  echo "Installed pre-push"
fi

#!/bin/sh
# Install Git hooks: symlink .git/hooks/* to scripts/hooks/*.sh
# Run from repo root. Idempotent.
set -e

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
HOOKS_DIR=".git/hooks"
SCRIPTS_DIR="scripts/hooks"

mkdir -p "$HOOKS_DIR"

# pre-commit: go build, ws-verdict validation
if [ -f "$SCRIPTS_DIR/pre-commit.sh" ]; then
  ln -sf ../../scripts/hooks/pre-commit.sh "$HOOKS_DIR/pre-commit"
  chmod +x "$SCRIPTS_DIR/pre-commit.sh"
  echo "Installed pre-commit"
fi

# pre-push: go test -short, evidence validation for feature branches
if [ -f "$SCRIPTS_DIR/pre-push.sh" ]; then
  ln -sf ../../scripts/hooks/pre-push.sh "$HOOKS_DIR/pre-push"
  chmod +x "$SCRIPTS_DIR/pre-push.sh"
  echo "Installed pre-push"
fi

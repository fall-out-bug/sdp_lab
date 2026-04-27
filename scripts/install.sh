#!/usr/bin/env bash
# F141-03: one-shot SDP installer for downstream repos.
# Usage: curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
#
# Environment overrides:
#   SDP_REPO    GitHub repo slug (default: fall-out-bug/sdp_lab)
#   SDP_BRANCH  Branch/tag to clone (default: main)
#   SDP_HARNESS Harness selection: auto|all|claude-code,opencode,... (default: auto)
#   SDP_TARGET  Target directory (default: .)
set -euo pipefail

REPO="${SDP_REPO:-fall-out-bug/sdp_lab}"
BRANCH="${SDP_BRANCH:-main}"
HARNESS="${SDP_HARNESS:-auto}"
TARGET="${SDP_TARGET:-.}"

echo "→ SDP installer: harness=$HARNESS target=$TARGET"

# Detect platform
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)       ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
esac

echo "→ platform: $OS/$ARCH"

# Strategy 1: if `sdp` binary already on PATH, use it directly.
if command -v sdp >/dev/null 2>&1; then
  echo "→ found sdp binary on PATH: $(command -v sdp)"
  sdp init --harness "$HARNESS" --target "$TARGET"
  echo "✓ SDP installed in $TARGET"
  exit 0
fi

# Strategy 2: clone-and-build (offline-friendly, no GitHub Releases needed in v1).
# Requires: git, go (1.21+)
for dep in git go; do
  if ! command -v "$dep" >/dev/null 2>&1; then
    echo "error: required tool '$dep' not found on PATH" >&2
    echo "       Please install $dep and re-run this script." >&2
    exit 1
  fi
done

TMPDIR_SDP="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_SDP"' EXIT

echo "→ cloning $REPO@$BRANCH into $TMPDIR_SDP/sdp_lab"
git clone --depth=1 --branch "$BRANCH" "https://github.com/$REPO.git" "$TMPDIR_SDP/sdp_lab" 2>&1

echo "→ building sdp binary"
(cd "$TMPDIR_SDP/sdp_lab" && go build -tags "sqlite_fts5" -o "$TMPDIR_SDP/sdp" ./cmd/sdp 2>&1)

echo "→ running sdp init"
"$TMPDIR_SDP/sdp" init --harness "$HARNESS" --target "$TARGET"

echo "✓ SDP installed in $TARGET"

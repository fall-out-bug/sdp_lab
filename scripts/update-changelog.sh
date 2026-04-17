#!/usr/bin/env bash
# update-changelog.sh — Auto-generate CHANGELOG.md entry from recent commits.
# Called from CI on push to main. Non-blocking: always exits 0.
set -euo pipefail

echo "::group::CHANGELOG auto-generation"

# Run the changelog update via the sdp-doc-sync CLI.
# Uses the default range (HEAD~1..HEAD) unless overridden by SDP_SINCE env var.
if ! go run ./cmd/sdp-doc-sync --mode changelog ${SDP_SINCE:+--since "$SDP_SINCE"}; then
  echo "WARN: changelog generation failed (non-blocking)"
  echo "::endgroup::"
  exit 0
fi

# Check if docs/CHANGELOG.md was actually modified.
if ! git diff --quiet docs/CHANGELOG.md 2>/dev/null; then
  echo "CHANGELOG.md updated — committing"

  git config user.name "github-actions[bot]"
  git config user.email "github-actions[bot]@users.noreply.github.com"
  git add docs/CHANGELOG.md
  git commit -m "chore(changelog): auto-update [skip ci]"

  # Push only if we have a token (CI context).
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    git push origin HEAD:"${GITHUB_REF_NAME:-main}" || {
      echo "WARN: changelog push failed (non-blocking)"
    }
  else
    echo "INFO: no GITHUB_TOKEN — skipping push (local run)"
  fi
else
  echo "No CHANGELOG.md changes detected"
fi

echo "::endgroup::"
exit 0

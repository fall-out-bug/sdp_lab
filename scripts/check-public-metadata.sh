#!/usr/bin/env bash
# check-public-metadata.sh — Detect stale URLs and org/repo references in user-facing files.
#
# F078-02: Metadata and links drift CI gate
#
# Scope: user-facing docs and manifests only (not Go code, not archive docs).
#
# Checks:
#   1. Template URLs with unreplaced placeholders (OWNER/REPO, YOUR_ORG)
#   2. Invalid/malformed GitHub URLs
#   3. Legacy release download URL patterns (public sdp repo in sdp_lab install/release URLs)
#   4. Installer raw URL consistency
#
# Usage:
#   scripts/check-public-metadata.sh              # human-readable output
#   scripts/check-public-metadata.sh --json       # machine-readable JSON
#
# Exit codes:
#   0 — no issues found
#   1 — stale references or broken URLs detected
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

JSON_MODE=false
[[ "${1:-}" == "--json" ]] && JSON_MODE=true

# --- User-facing files only ---
USER_FACING_FILES=(
  README.md
  CONTRIBUTING.md
  sdp.manifest.yaml
)

# Add docs/ top-level .md files (excluding internal/project docs)
while IFS= read -r f; do
  USER_FACING_FILES+=("$f")
done < <(find docs -maxdepth 1 -name "*.md" -not -name "PROJECT_*" -not -name "UP_*" -not -name "AGENT_*" -not -name "ARTIFACT_*" -not -name "OBSERVABILITY_*" 2>/dev/null || true)

# --- Accumulators ---
FINDINGS=()

# --- Check 1: Template URL placeholders ---
check_templates() {
  local patterns=("OWNER/REPO" "YOUR_ORG" "YOUR_REPO" "your-org" "your-repo")
  for f in "${USER_FACING_FILES[@]}"; do
    [ -f "$f" ] || continue
    for pat in "${patterns[@]}"; do
      while IFS= read -r match; do
        [ -z "$match" ] && continue
        local lineno="${match%%:*}"
        FINDINGS+=("template-url:$f:$lineno:unreplaced placeholder '$pat'")
      done < <(grep -n "$pat" "$f" 2>/dev/null || true)
    done
  done
}

# --- Check 2: Legacy release URLs referencing the public sdp mirror ---
check_legacy_downloads() {
  for f in "${USER_FACING_FILES[@]}"; do
    [ -f "$f" ] || continue
    # Deduplicate by line number within each file
    declare -A seen_lines
    while IFS= read -r match; do
      [ -z "$match" ] && continue
      local lineno="${match%%:*}"
      local key="$f:$lineno"
      if [ -z "${seen_lines[$key]:-}" ]; then
        seen_lines[$key]=1
        FINDINGS+=("legacy-url:$f:$lineno:release download URL references public sdp mirror; sdp_lab release examples must use sdp_lab")
      fi
    done < <(grep -nE 'github\.com/fall-out-bug/sdp/releases/download' "$f" 2>/dev/null || true)
    unset seen_lines
  done
}

# --- Check 3: Malformed URLs ---
check_malformed_urls() {
  for f in "${USER_FACING_FILES[@]}"; do
    [ -f "$f" ] || continue
    # Double slashes in github.com URLs (not https://)
    while IFS= read -r match; do
      [ -z "$match" ] && continue
      local lineno="${match%%:*}"
      FINDINGS+=("malformed-url:$f:$lineno:double slash in GitHub URL")
    done < <(grep -n 'github\.com//[^/]' "$f" 2>/dev/null || true)
    # Missing repo name after org
    while IFS= read -r match; do
      [ -z "$match" ] && continue
      local lineno="${match%%:*}"
      FINDINGS+=("malformed-url:$f:$lineno:GitHub URL missing repo name")
    done < <(grep -nP 'github\.com/fall-out-bug/?[ "\n]' "$f" 2>/dev/null || true)
  done
}

# --- Check 4: Installer raw URL consistency ---
check_url_consistency() {
  for f in "${USER_FACING_FILES[@]}"; do
    [ -f "$f" ] || continue
    while IFS= read -r match; do
      [ -z "$match" ] && continue
      local lineno="${match%%:*}"
      FINDINGS+=("url-drift:$f:$lineno:installer URL references public sdp mirror; sdp_lab onboarding must install from sdp_lab")
    done < <(grep -n 'raw\.githubusercontent\.com/fall-out-bug/sdp/main/scripts/install\.sh' "$f" 2>/dev/null || true)
  done
}

# --- Run checks ---
check_templates
check_legacy_downloads
check_malformed_urls
check_url_consistency

# --- Output ---
TOTAL=${#FINDINGS[@]}

if $JSON_MODE; then
  findings_json="[]"
  if [ "$TOTAL" -gt 0 ]; then
    findings_json=$(printf '%s\n' "${FINDINGS[@]}" | jq -R -c 'split(":") as $parts | try {severity: $parts[0], file: $parts[1], line: ($parts[2] | tonumber), message: ($parts[3:] | join(":"))} catch {severity: $parts[0], file: $parts[1], line: 0, message: ($parts[2:] | join(":"))}' | jq -s -c '.')
  fi
  echo "{\"ok\":$([ "$TOTAL" -eq 0 ] && echo true || echo false),\"finding_count\":$TOTAL,\"findings\":$findings_json}"
else
  echo "=== Public Metadata Drift Check ==="
  echo ""
  if [ "$TOTAL" -eq 0 ]; then
    echo "RESULT: OK — 0 stale references found in user-facing files"
  else
    echo "RESULT: DRIFT — $TOTAL stale reference(s) found"
    echo ""
    for f in "${FINDINGS[@]}"; do
      severity=$(echo "$f" | cut -d: -f1)
      file=$(echo "$f" | cut -d: -f2)
      lineno=$(echo "$f" | cut -d: -f3)
      msg=$(echo "$f" | cut -d: -f4-)
      echo "  [$severity] $file:$lineno — $msg"
    done
    echo ""
    echo "Local reproduce: scripts/check-public-metadata.sh"
  fi
fi

if [ "$TOTAL" -gt 0 ]; then
  exit 1
fi

exit 0

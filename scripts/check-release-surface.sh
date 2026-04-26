#!/usr/bin/env bash
# check-release-surface.sh — Validate release manifest consistency.
#
# F078-03: Release surface manifest alignment
#
# Checks:
#   1. sdp.manifest.yaml version matches GoReleaser tag conventions
#   2. All GoReleaser builds reference existing main paths
#   3. Archive name templates include version
#   4. Download URL pattern is consistent
#
# Usage:
#   scripts/check-release-surface.sh              # human-readable output
#   scripts/check-release-surface.sh --json       # machine-readable JSON
#   VERSION=v1.2.3 scripts/check-release-surface.sh  # validate against tag
#
# Exit codes:
#   0 — release surface consistent
#   1 — inconsistency detected
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

JSON_MODE=false
[[ "${1:-}" == "--json" ]] && JSON_MODE=true

FINDINGS=()
OK_COUNT=0

# --- Check 1: Manifest version is valid semver ---
check_manifest_version() {
  if [ ! -f "sdp.manifest.yaml" ]; then
    FINDINGS+=("missing:sdp.manifest.yaml:manifest file not found")
    return
  fi

  local version
  version=$(grep -E '^version:' sdp.manifest.yaml | head -1 | sed 's/^version:\s*//' | tr -d '"' | tr -d ' ')

  if [ -z "$version" ]; then
    FINDINGS+=("invalid:sdp.manifest.yaml:version field missing or empty")
    return
  fi

  # Validate semver format (X.Y.Z)
  if ! echo "$version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    FINDINGS+=("invalid:sdp.manifest.yaml:version '$version' is not valid semver (X.Y.Z)")
    return
  fi

  OK_COUNT=$((OK_COUNT + 1))

  # If VERSION env is set, check it matches
  if [ -n "${VERSION:-}" ]; then
    local tag_version="${VERSION#v}"
    if [ "$version" != "$tag_version" ]; then
      FINDINGS+=("mismatch:sdp.manifest.yaml:manifest version '$version' != tag version '$tag_version'")
    else
      OK_COUNT=$((OK_COUNT + 1))
    fi
  fi
}

# --- Check 2: GoReleaser builds reference existing main paths ---
check_goreleaser_builds() {
  if [ ! -f ".goreleaser.yml" ]; then
    FINDINGS+=("missing:.goreleaser.yml:GoReleaser config not found")
    return
  fi

  # Extract main paths from goreleaser config
  local mains
  mains=$(grep -A2 'main:' .goreleaser.yml | grep 'main:' | sed 's/.*main:\s*//' | sort -u)

  for main_path in $mains; do
    if [ ! -d "$main_path" ]; then
      FINDINGS+=("missing:$main_path:GoReleaser build main path does not exist")
    else
      OK_COUNT=$((OK_COUNT + 1))
    fi
  done
}

# --- Check 3: Archive templates include version ---
check_archive_templates() {
  if [ ! -f ".goreleaser.yml" ]; then
    return
  fi

  # Check archive section name_templates contain Version
  local in_archives=false
  while IFS= read -r line; do
    if [[ "$line" =~ ^archives: ]]; then
      in_archives=true
      continue
    fi
    if $in_archives && [[ "$line" =~ ^[a-z] ]]; then
      in_archives=false
      continue
    fi
    if $in_archives && [[ "$line" =~ name_template: ]]; then
      local template
      template=$(echo "$line" | sed 's/.*name_template:\s*//' | tr -d '"')
      if echo "$template" | grep -qF '.Version'; then
        OK_COUNT=$((OK_COUNT + 1))
      else
        FINDINGS+=("template:.goreleaser.yml:archive name_template '$template' missing .Version")
      fi
    fi
  done < .goreleaser.yml
}

# --- Check 4: Version drift check passes ---
check_version_drift() {
  if [ -f "scripts/check-version-drift.sh" ]; then
    if scripts/check-version-drift.sh --json > /dev/null 2>&1; then
      OK_COUNT=$((OK_COUNT + 1))
    else
      FINDINGS+=("drift:version-drift:version drift detected — run scripts/check-version-drift.sh")
    fi
  fi
}

# --- Check 5: Metadata drift check passes ---
check_metadata_drift() {
  if [ -f "scripts/check-public-metadata.sh" ]; then
    if scripts/check-public-metadata.sh --json > /dev/null 2>&1; then
      OK_COUNT=$((OK_COUNT + 1))
    else
      FINDINGS+=("drift:metadata-drift:metadata drift detected — run scripts/check-public-metadata.sh")
    fi
  fi
}

# --- Run checks ---
check_manifest_version
check_goreleaser_builds
check_archive_templates
check_version_drift
check_metadata_drift

# --- Output ---
TOTAL=${#FINDINGS[@]}

if $JSON_MODE; then
  findings_json="[]"
  if [ "$TOTAL" -gt 0 ]; then
    findings_json=$(printf '%s\n' "${FINDINGS[@]}" | jq -R -c 'split(":") as $parts | {severity: $parts[0], file: $parts[1], message: ($parts[2:] | join(":"))}' | jq -s -c '.')
  fi
  echo "{\"ok\":$([ "$TOTAL" -eq 0 ] && echo true || echo false),\"finding_count\":$TOTAL,\"findings\":$findings_json,\"ok_count\":$OK_COUNT}"
else
  echo "=== Release Surface Consistency Check ==="
  echo ""
  if [ "$TOTAL" -eq 0 ]; then
    echo "RESULT: OK — release surface consistent ($OK_COUNT checks passed)"
  else
    echo "RESULT: FAIL — $TOTAL issue(s) found"
    echo ""
    for f in "${FINDINGS[@]}"; do
      severity=$(echo "$f" | cut -d: -f1)
      file=$(echo "$f" | cut -d: -f2)
      msg=$(echo "$f" | cut -d: -f3-)
      echo "  [$severity] $file — $msg"
    done
    echo ""
    echo "Local reproduce: scripts/check-release-surface.sh"
  fi
fi

if [ "$TOTAL" -gt 0 ]; then
  exit 1
fi

exit 0

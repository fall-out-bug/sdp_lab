#!/usr/bin/env bash
# check-version-drift.sh — Verify all version declarations match the canonical source.
#
# Canonical source: sdp.manifest.yaml → version field
#
# Usage:
#   scripts/check-version-drift.sh              # human-readable output
#   scripts/check-version-drift.sh --json       # machine-readable JSON
#
# Exit codes:
#   0 — all versions consistent
#   1 — drift detected (or fatal error)
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

JSON_MODE=false
[[ "${1:-}" == "--json" ]] && JSON_MODE=true

# --- Read canonical version from sdp.manifest.yaml ---
CANONICAL=""
if [ -f "sdp.manifest.yaml" ]; then
  CANONICAL=$(grep -E '^version:' sdp.manifest.yaml | head -1 | sed 's/^version:\s*//' | tr -d '"' | tr -d ' ')
fi

if [ -z "$CANONICAL" ]; then
  if $JSON_MODE; then
    echo '{"ok":false,"error":"canonical version not found in sdp.manifest.yaml"}'
  else
    echo "ERROR: canonical version not found in sdp.manifest.yaml"
  fi
  exit 1
fi

# --- Extraction helpers ---
extract_yaml_field() {
  local file="$1" field="$2"
  grep -E "^${field}:" "$file" | head -1 | sed "s/^${field}:\s*//" | tr -d '"' | tr -d ' '
}

extract_go_const() {
  local file="$1" pattern="$2"
  grep -oE "${pattern}" "$file" | grep -oE '"[^"]+"' | tr -d '"'
}

extract_go_assign() {
  local file="$1" pattern="$2"
  grep -oE "${pattern}" "$file" | head -1 | grep -oE '"[^"]+"' | tr -d '"'
}

# --- Accumulators ---
DRIFTS=()
OKS=()
SKIPS=()

check_release_version() {
  local file="$1" label="$2" value="$3"
  if [ -z "$value" ]; then
    SKIPS+=("$file ($label: not extractable)")
    return
  fi
  if [ "$value" != "$CANONICAL" ]; then
    DRIFTS+=("$file: expected '$CANONICAL', got '$value'")
  else
    OKS+=("$file")
  fi
}

# --- Release version surfaces (must match CANONICAL) ---

# 1. sdp.manifest.yaml → sdp_version
if [ -f "sdp.manifest.yaml" ]; then
  v=$(extract_yaml_field sdp.manifest.yaml "sdp_version")
  check_release_version "sdp.manifest.yaml[sdp_version]" "sdp_version" "$v"
else
  SKIPS+=("sdp.manifest.yaml (not found)")
fi

# 2. manifest template
if [ -f "cmd/sdp/templates/sdp.manifest.template.yaml" ]; then
  v=$(extract_yaml_field cmd/sdp/templates/sdp.manifest.template.yaml "version")
  check_release_version "cmd/sdp/templates/sdp.manifest.template.yaml[version]" "version" "$v"
else
  SKIPS+=("cmd/sdp/templates/sdp.manifest.template.yaml (not found)")
fi

# 3. bootstrap version constant
if [ -f "internal/bootstrap/bootstrap.go" ]; then
  v=$(extract_go_const internal/bootstrap/bootstrap.go 'const version = "[^"]+"')
  check_release_version "internal/bootstrap/bootstrap.go" "const version" "$v"
else
  SKIPS+=("internal/bootstrap/bootstrap.go (not found)")
fi

# 4. cmd_init.go sdpVersion default
if [ -f "cmd/sdp/cmd_init.go" ]; then
  v=$(extract_go_assign cmd/sdp/cmd_init.go 'sdpVersion = "[^"]+"')
  check_release_version "cmd/sdp/cmd_init.go" "sdpVersion" "$v"
else
  SKIPS+=("cmd/sdp/cmd_init.go (not found)")
fi

# 5. profile config (nested under profile: key)
if [ -f "configs/profiles/oss-combine/config.yaml" ]; then
  v=$(grep -A5 '^profile:' configs/profiles/oss-combine/config.yaml | grep 'version:' | head -1 | sed 's/.*version:\s*//' | tr -d '"' | tr -d ' ')
  check_release_version "configs/profiles/oss-combine/config.yaml" "profile.version" "$v"
else
  SKIPS+=("configs/profiles/oss-combine/config.yaml (not found)")
fi

# --- Protocol spec versions (independent of release version, must be internally consistent) ---
PROTOCOL_VERSIONS=()

if [ -f "internal/cli/status_view.go" ]; then
  v=$(extract_go_const internal/cli/status_view.go 'const statusSpecVersion = "[^"]+"')
  [ -n "$v" ] && PROTOCOL_VERSIONS+=("internal/cli/status_view.go:$v")
fi

if [ -f "internal/cli/instructions_view.go" ]; then
  v=$(extract_go_const internal/cli/instructions_view.go 'const instructionSpecVersion = "[^"]+"')
  [ -n "$v" ] && PROTOCOL_VERSIONS+=("internal/cli/instructions_view.go:$v")
fi

if [ -f "internal/control/control.go" ]; then
  v=$(extract_go_assign internal/control/control.go 'specVersion\s+= "[^"]+"')
  [ -n "$v" ] && PROTOCOL_VERSIONS+=("internal/control/control.go:$v")
fi

# Check protocol versions consistency
PROTOCOL_BASE=""
PROTOCOL_DRIFT=false
for pv in "${PROTOCOL_VERSIONS[@]+"${PROTOCOL_VERSIONS[@]}"}"; do
  pv_version="${pv#*:}"
  pv_file="${pv%%:*}"
  if [ -z "$PROTOCOL_BASE" ]; then
    PROTOCOL_BASE="$pv_version"
  elif [ "$pv_version" != "$PROTOCOL_BASE" ]; then
    PROTOCOL_DRIFT=true
    DRIFTS+=("$pv_file: protocol drift, expected '$PROTOCOL_BASE', got '$pv_version'")
  fi
done

# --- Output ---
TOTAL=${#DRIFTS[@]}
OK_COUNT=${#OKS[@]}
SKIP_COUNT=${#SKIPS[@]}

if $JSON_MODE; then
  drift_arr="[]"
  if [ "$TOTAL" -gt 0 ]; then
    drift_arr=$(printf '%s\n' "${DRIFTS[@]}" | jq -R -s -c 'split("\n") | map(select(length > 0))')
  fi
  protocol_arr="[]"
  if [ "${#PROTOCOL_VERSIONS[@]}" -gt 0 ]; then
    protocol_arr=$(printf '%s\n' "${PROTOCOL_VERSIONS[@]}" | jq -R -s -c 'split("\n") | map(select(length > 0))')
  fi
  echo "{\"ok\":$([ "$TOTAL" -eq 0 ] && echo true || echo false),\"canonical\":\"$CANONICAL\",\"drift_count\":$TOTAL,\"drifts\":$drift_arr,\"ok_count\":$OK_COUNT,\"skip_count\":$SKIP_COUNT,\"protocol_versions\":$protocol_arr,\"protocol_drift\":$PROTOCOL_DRIFT}"
else
  echo "=== Version Drift Check ==="
  echo "Canonical: $CANONICAL (sdp.manifest.yaml)"
  echo ""
  echo "--- Release version surfaces (must match '$CANONICAL') ---"
  for o in "${OKS[@]+"${OKS[@]}"}"; do
    echo "  OK:    $o"
  done
  for s in "${SKIPS[@]+"${SKIPS[@]}"}"; do
    echo "  SKIP:  $s"
  done
  for d in "${DRIFTS[@]+"${DRIFTS[@]}"}"; do
    echo "  DRIFT: $d"
  done
  echo ""
  echo "--- Protocol spec versions (must be internally consistent) ---"
  for pv in "${PROTOCOL_VERSIONS[@]+"${PROTOCOL_VERSIONS[@]}"}"; do
    echo "  OK:    ${pv%%:*} → '${pv#*:}'"
  done
  echo ""
  if [ "$TOTAL" -eq 0 ]; then
    echo "RESULT: OK — $OK_COUNT surfaces verified, $SKIP_COUNT skipped, 0 drifts"
  else
    echo "RESULT: DRIFT — $TOTAL drift(s) detected across $OK_COUNT verified surfaces"
  fi
fi

if [ "$TOTAL" -gt 0 ]; then
  exit 1
fi

exit 0

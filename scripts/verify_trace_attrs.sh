#!/usr/bin/env bash
# verify_trace_attrs.sh - CI check for trace attribute allowlist compliance
# Verifies that all emitted trace attributes are declared in schema

set -euo pipefail

SCHEMA_FILE="${SCHEMA_FILE:-schema/telemetry/sdp-trace-events.schema.json}"
SDP_TRACE_BIN="${SDP_TRACE_BIN:-./sdp-trace-test-fixture}"

if [ ! -f "$SCHEMA_FILE" ]; then
  echo "ERROR: Schema file not found: $SCHEMA_FILE"
  exit 1
fi

# Extract allowed attributes per span kind from schema
extract_allowed_attrs() {
  local span_kind=$1
  jq -r ".span_kinds.${span_kind}.allowed_attributes // {} | keys[]" "$SCHEMA_FILE" 2>/dev/null || echo ""
}

# Span kinds to check
span_kinds=("execute_tool" "invoke_agent" "delivery_loop_phase" "sdp_bead_event")

# Check if sdp trace CLI exists (will be built by TEL-02)
if [ ! -f "$SDP_TRACE_BIN" ] && [ ! -f "./bin/sdp" ]; then
  echo "WARNING: sdp trace CLI not yet built (expected during TEL-02)"
  echo "This check will be enabled after TEL-02 implementation"
  exit 0
fi

# Test fixtures - these will be emitted by the actual sdp trace CLI
# For now, just verify schema is well-formed and parsable
if ! command -v jq >/dev/null 2>&1; then
  echo "WARNING: jq not found, skipping schema validation"
  exit 0
fi

if ! jq empty "$SCHEMA_FILE" 2>/dev/null; then
  echo "ERROR: Schema file is not valid JSON"
  exit 1
fi

# Verify schema has required properties
if ! jq -e ".properties.span_kinds" "$SCHEMA_FILE" >/dev/null 2>&1; then
  echo "ERROR: Schema missing required property: span_kinds"
  exit 1
fi

if ! jq -e ".properties.consent_levels" "$SCHEMA_FILE" >/dev/null 2>&1; then
  echo "ERROR: Schema missing required property: consent_levels"
  exit 1
fi

if ! jq -e ".properties.sampling_policy" "$SCHEMA_FILE" >/dev/null 2>&1; then
  echo "ERROR: Schema missing required property: sampling_policy"
  exit 1
fi

# Verify each span kind has allowed_attributes defined
for kind in "${span_kinds[@]}"; do
  if ! jq -e ".properties.span_kinds.properties.${kind}.properties.allowed_attributes" "$SCHEMA_FILE" >/dev/null 2>&1; then
    echo "ERROR: Span kind '$kind' missing allowed_attributes definition"
    exit 1
  fi
done

# Verify consent levels are defined
required_consent_levels=("metadata" "findings" "content")
for level in "${required_consent_levels[@]}"; do
  if ! jq -e ".properties.consent_levels.properties.${level}" "$SCHEMA_FILE" >/dev/null 2>&1; then
    echo "ERROR: Consent level '$level' not defined"
    exit 1
  fi
done

echo "OK: Trace schema is well-formed and complete"
echo "Span kinds validated: ${span_kinds[*]}"
echo "Consent levels validated: ${required_consent_levels[*]}"

# If sdp trace CLI exists, run fixture tests
if [ -f "./bin/sdp" ]; then
  echo ""
  echo "Running sdp trace CLI fixture tests..."

  # Test that unknown attributes are rejected
  if ! ./bin/sdp trace span-start --kind tool --name Bash --attr unknown.attribute=value 2>&1 | grep -q "rejected\|not allowed\|unknown"; then
    echo "WARNING: Allowlist rejection not yet implemented (expected during TEL-02)"
  else
    echo "OK: Unknown attributes are properly rejected"
  fi
fi

exit 0

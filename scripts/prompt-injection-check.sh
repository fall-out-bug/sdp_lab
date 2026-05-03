#!/usr/bin/env bash
# =============================================================================
# F164-05: Prompt-injection boundary claim check
#
# Grep/eval check that fails (exit 1) if any active SDP prompt surface
# claims prompt-only isolation is a security boundary.
#
# PI-013 supply-chain check: prompt bundles must not claim that formatting
# or text isolation alone constitutes a security control. Trust must
# come from deterministic policy, trust labels, and phase tool allowlists
# per F164 threat model.
#
# Usage:
#   ./scripts/prompt-injection-check.sh            # all surfaces
#   ./scripts/prompt-injection-check.sh prompts/   # scoped
#   ./scripts/prompt-injection-check.sh .agents/  # scoped
#
# Refs:
#   docs/security/f164-prompt-injection-threat-model.md
#   docs/security/f164-prompt-injection-test-cases.md (PI-013)
# =============================================================================

set -euo pipefail

SCOPE="${1:-prompts/ .agents/skills/ .pi/APPEND_SYSTEM.md docs/reference/skills.md}"
FOUND=0

# Patterns that claim prompt-only isolation is a security boundary.
# Phrases like "this prompt is a security boundary" or "instruction isolation
# is the security control" are PI-013 failures. Exceptions are made for:
#   - advisory linking to F164 corpus ("see F164 PI-013")
#   - "must NOT rely on prompt-only isolation as a security boundary"
#     (this is the correct pattern — flagging it would be a false positive)
PATTERNS=(
  # Claim that prompt formatting/structure is a security control
  "prompt.isolation.is.a.security.control"
  "prompt.boundary.*security.control"
  "security.control.*prompt.isolation"
  "instruction.isolation.is.the.security.control"
  "isolated.prompt.as.a.security.boundary"

  # Claim that prompt-only protection suffices
  "prompt.only.protection.is.sufficient"
  "sufficient.as.a.security.boundary"
  "no.additional.controls.needed"
)

echo "=== F164 PI-013: Prompt-only protection is NOT a security boundary ==="
echo "Checking: $SCOPE"
echo ""

for pattern in "${PATTERNS[@]}"; do
  # Grep returns 0 (match) or 1 (no match). We only care about matches.
  # shellcheck disable=SC2086
  matches=$(grep -rE "$pattern" $SCOPE 2>/dev/null || true)
  if [ -n "$matches" ]; then
    echo "FAIL: Found claim that prompt-only isolation is a security boundary:"
    echo "$matches"
    echo ""
    FOUND=$((FOUND + 1))
  fi
done

# Counter-example: the CORRECT advisory pattern should NOT trigger.
# Double-check we are not falsely flagging legitimate F164 guidance:
echo "Checking counter-example (should pass — F164 advisory):"
# This pattern in F164 docs is the RIGHT way to say it.
counter_check=$(grep -rE "must NOT rely on prompt.only isolation as a security boundary" "$SCOPE" 2>/dev/null || true)
if [ -n "$counter_check" ]; then
  echo "  OK: Found correct advisory pattern (not a PI-013 failure):"
  echo "$counter_check"
  echo ""
fi

if [ $FOUND -gt 0 ]; then
  echo "============================================"
  echo "FAIL: $FOUND prompt surface(s) claim prompt-only protection is a security boundary."
  echo "Per F164 PI-013, trust must come from deterministic policy, trust labels,"
  echo "and phase tool allowlists — not from prompt text isolation alone."
  echo ""
  echo "Reference: docs/security/f164-prompt-injection-threat-model.md"
  echo "           docs/security/f164-prompt-injection-test-cases.md (PI-013)"
  exit 1
else
  echo "PASS: No active prompt surface claims prompt-only isolation is a security boundary."
  echo "Per F164 PI-013, trust comes from deterministic policy, trust labels, and phase allowlists."
  exit 0
fi
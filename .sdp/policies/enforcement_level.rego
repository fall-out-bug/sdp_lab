package sdp.policies

# Enforcement level controls whether policy denials block merges or are advisory only.
#
# Values:
#   "advisory"  - gates report violations but do not block
#   "blocking"  - gates block merge on violations
#
# Graduation plan (A* stream):
#   - 2026-03-10: keep advisory while CI-001..CI-003 roll out and stabilize
#   - 2026-03-17: switch CI variable SDP_POLICY_ENFORCEMENT_MODE=blocking
#
# Enforcement mode is now explicit input from CI, so policy behavior is not
# permanently advisory.
#
# This single value governs all SDP policies. The deny rules in other
# policy files use `effective_deny` so they respect this level.

default enforcement_level = "advisory"

enforcement_level = "blocking" {
    input.enforcement_mode == "blocking"
}

# effective_deny is the final list of denials to act on.
# In advisory mode, violations are logged but empty (no block).
# In blocking mode, violations are surfaced for enforcement.
effective_deny[msg] {
    enforcement_level == "blocking"
    deny[msg]
}

# advisory_warn surfaces violations as warnings regardless of enforcement level.
advisory_warn[msg] {
    deny[msg]
}

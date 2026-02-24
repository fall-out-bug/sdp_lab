package sdp.policies

# Enforcement level controls whether policy denials block merges or are advisory only.
#
# Values:
#   "advisory"  - gates report violations but do not block (Phase 8A default)
#   "blocking"  - gates block merge on violations (flip when ready)
#
# To enable blocking enforcement:
#   Change default enforcement_level to "blocking"
#   Commit the change — it becomes effective on the next PR.
#
# This single value governs all SDP policies. The deny rules in other
# policy files use `effective_deny` so they respect this level.

default enforcement_level = "advisory"

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

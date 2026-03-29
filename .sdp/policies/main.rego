package sdp.policies

import future.keywords.if
import future.keywords.in
import future.keywords.contains

# Enforcement level from input (default: advisory)
enforcement_level := input.enforcement_mode if {
	input.enforcement_mode
}

enforcement_level := "advisory" if {
	not input.enforcement_mode
}

# === DENY RULES (block merge when enforcement=blocking) ===

# P0 findings must be zero
deny contains msg if {
	input.p0_findings > 0
	msg := sprintf("P0 findings present (%d) — must be resolved before merge", [input.p0_findings])
}

# Evidence validation must pass if evidence files exist
deny contains msg if {
	input.evidence_files_count > 0
	not input.evidence_validation_passed
	msg := "evidence validation failed — attestation files are invalid"
}

# Scope violations block merge
deny contains msg if {
	input.scope_violations_count > 0
	msg := sprintf("scope violations detected (%d) — code changes outside declared workstream boundary", [input.scope_violations_count])
}

# Feature changes must reference beads issues
deny contains msg if {
	input.has_feature_changes
	not input.beads_referenced
	msg := "feature changes must reference a beads issue in commit messages"
}

# === WARN RULES (always reported, never block) ===

warn contains msg if {
	input.p1_findings > 0
	msg := sprintf("P1 findings present (%d) — should be addressed before merge", [input.p1_findings])
}

warn contains msg if {
	input.p2_findings > 0
	msg := sprintf("P2 findings present (%d) — consider addressing", [input.p2_findings])
}

warn contains msg if {
	input.has_feature_changes
	input.evidence_files_count == 0
	msg := "feature changes without evidence files — consider running auto-attestation"
}

# === EFFECTIVE OUTPUTS ===

# effective_deny: only enforce denials when mode=blocking
effective_deny := deny if {
	enforcement_level == "blocking"
}

effective_deny := [] if {
	enforcement_level != "blocking"
}

# advisory_warn: always show warnings + denials in advisory mode
advisory_warn := warn if {
	enforcement_level == "blocking"
}

advisory_warn := array.concat(
	[msg | some msg in deny],
	[msg | some msg in warn],
) if {
	enforcement_level != "blocking"
}

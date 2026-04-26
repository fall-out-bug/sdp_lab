package sdp.evidence_gate

import rego.v1

# Default configuration for evidence gate thresholds.
# Override via input.config to customize per-deployment.

default allow := false

allow if {
	input.attestation.signed == true
	not has_denied_severities
	verification_passed == true
}

# Severity threshold checks (configurable via input.config.thresholds)
has_denied_severities if {
	severities := {"critical", "high", "medium", "low"}
	s := severities[_]
	exceeds_threshold(s)
}

exceeds_threshold(severity) if {
	threshold := get_threshold(severity)
	actual := get_actual_count(severity)
	actual > threshold
}

get_threshold(severity) := input.config.thresholds[severity] if {
	input.config.thresholds[severity]
} else := default_threshold(severity)

default_threshold("critical") := 0
default_threshold("high") := 0
default_threshold("medium") := 5
default_threshold("low") := 20

get_actual_count(severity) := count([d | d := input.discrepancies[_]; strings.lower(d.severity) == severity])

verification_passed if {
	input.attestation.verified == true
} else := false

# Deny reasons for audit trail
deny_reasons contains msg if {
	input.attestation.signed != true
	msg := "attestation is not signed"
}

deny_reasons contains msg if {
	verification_passed != true
	msg := sprintf("attestation verification failed: %s", [input.attestation.verification_error])
}

deny_reasons contains msg if {
	severities := {"critical", "high", "medium", "low"}
	severity := severities[_]
	exceeds_threshold(severity)
	msg := sprintf("%s discrepancies exceeded threshold: %d > %d", [severity, get_actual_count(severity), get_threshold(severity)])
}

# Audit output — included in audit bundle
audit_result := {
	"allowed": allow,
	"reasons": deny_reasons,
	"severity_count": {
		"critical": get_actual_count("critical"),
		"high": get_actual_count("high"),
		"medium": get_actual_count("medium"),
		"low": get_actual_count("low"),
	},
	"thresholds": {
		"critical": get_threshold("critical"),
		"high": get_threshold("high"),
		"medium": get_threshold("medium"),
		"low": get_threshold("low"),
	},
}

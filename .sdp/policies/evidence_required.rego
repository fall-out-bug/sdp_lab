package sdp.policies

default evidence_valid = false

evidence_valid {
    input.evidence_files_count > 0
    input.evidence_validation_passed == true
}

evidence_valid {
    input.evidence_files_count == 0
    not input.has_workstream_changes
}

deny[msg] {
    input.has_workstream_changes
    input.evidence_files_count == 0
    msg := "PR changes workstream files but has no evidence attestation"
}

deny[msg] {
    input.evidence_files_count > 0
    input.evidence_validation_passed == false
    msg := "Evidence attestation present but validation failed"
}

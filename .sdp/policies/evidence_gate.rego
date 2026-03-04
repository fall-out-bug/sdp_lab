package sdp.policies

default evidence_gate_passed = true

evidence_gate_passed {
    att := object.get(input, "attestation", {})
    object.get(att, "require_signed", true) == false
}

evidence_gate_passed {
    att := object.get(input, "attestation", {})
    object.get(att, "require_signed", true) == true
    object.get(att, "signed", false) == true
}

deny[msg] {
    att := object.get(input, "attestation", {})
    object.get(att, "require_signed", true) == true
    object.get(att, "signed", false) == false
    msg := "Evidence gate requires a signed attestation"
}

deny[msg] {
    counts := object.get(object.get(input, "discrepancies", {}), "counts", {})
    thresholds := object.get(object.get(input, "discrepancies", {}), "thresholds", {})
    critical_count := object.get(counts, "critical", 0)
    critical_max := object.get(thresholds, "critical", 0)
    critical_count > critical_max
    msg := sprintf("Critical discrepancies exceed threshold (%d > %d)", [critical_count, critical_max])
}

deny[msg] {
    counts := object.get(object.get(input, "discrepancies", {}), "counts", {})
    thresholds := object.get(object.get(input, "discrepancies", {}), "thresholds", {})
    high_count := object.get(counts, "high", 0)
    high_max := object.get(thresholds, "high", 0)
    high_count > high_max
    msg := sprintf("High discrepancies exceed threshold (%d > %d)", [high_count, high_max])
}

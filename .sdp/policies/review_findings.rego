package sdp.policies

default no_blocking_findings = true

no_blocking_findings = false {
    input.p0_findings > 0
}

no_blocking_findings = false {
    input.p1_findings > 0
}

deny[msg] {
    input.p0_findings > 0
    msg := sprintf("P0 findings block merge: %d open", [input.p0_findings])
}

deny[msg] {
    input.p1_findings > 0
    msg := sprintf("P1 findings block merge: %d open", [input.p1_findings])
}

warn[msg] {
    input.p2_findings > 0
    msg := sprintf("P2 findings should be tracked in beads: %d open", [input.p2_findings])
}

package sdp.policies

default scope_compliant = true

scope_compliant = false {
    input.scope_violations_count > 0
}

deny[msg] {
    input.scope_violations_count > 0
    msg := sprintf("Scope violations detected: %d files outside declared scope", [input.scope_violations_count])
}

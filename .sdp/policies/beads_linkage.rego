package sdp.policies

default beads_linked = true

beads_linked = false {
    input.has_feature_changes
    input.beads_referenced == false
}

deny[msg] {
    input.has_feature_changes
    input.beads_referenced == false
    msg := "Feature changes without beads issue reference in commits"
}

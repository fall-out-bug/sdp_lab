package dispatch

// SelectTiers groups profiles into ordered tier-chains for cascade routing.
//
// Given a Router-ranked slice of profiles and an ordered list of TierClass
// values, returns a slice-of-slices where index i contains all profiles whose
// TierClass matches order[i]. Profiles with empty or unrecognised TierClass
// land in a final "untiered" bucket appended after the requested order.
//
// The relative order within each tier is preserved from the input (so a
// caller-supplied ranking is respected within the tier).
func SelectTiers(profiles []*CapabilityProfile, order []TierClass) [][]*CapabilityProfile {
	out := make([][]*CapabilityProfile, len(order)+1) // +1 for untiered
	for i := range out {
		out[i] = []*CapabilityProfile{}
	}
	tierIdx := make(map[TierClass]int, len(order))
	for i, t := range order {
		tierIdx[t] = i
	}
	untiered := len(order)
	for _, p := range profiles {
		idx, ok := tierIdx[p.TierClass]
		if !ok {
			idx = untiered
		}
		out[idx] = append(out[idx], p)
	}
	return out
}

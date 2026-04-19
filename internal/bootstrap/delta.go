package bootstrap

import "sort"

// DeltaType constants follow OpenSpec-style change markers.
const (
	DeltaAdded    = "ADDED"
	DeltaModified = "MODIFIED"
	DeltaRemoved  = "REMOVED"
)

// Delta represents a single change between existing project rules and fresh
// scout-derived rules.
type Delta struct {
	Section    string `json:"section"`            // "testing", "ci", "linting", "architecture", "language", etc.
	ChangeType string `json:"change_type"`        // ADDED, MODIFIED, REMOVED
	Old        string `json:"old,omitempty"`      // previous value (empty for ADDED)
	New        string `json:"new,omitempty"`       // current value (empty for REMOVED)
	Evidence   string `json:"evidence"`            // what scout data supports this change
}

// CompareSections compares existing project rule sections with fresh
// scout-derived sections and returns a sorted, deterministic list of Deltas.
// The result is sorted by Section name for reproducibility.
func CompareSections(existing, fresh map[string]string) []Delta {
	var deltas []Delta

	// Detect MODIFIED and REMOVED: sections present in existing.
	for section, oldVal := range existing {
		newVal, found := fresh[section]
		switch {
		case !found:
			deltas = append(deltas, Delta{
				Section:    section,
				ChangeType: DeltaRemoved,
				Old:        oldVal,
				Evidence:   "section present in existing rules but absent from fresh scout",
			})
		case found && oldVal != newVal:
			deltas = append(deltas, Delta{
				Section:    section,
				ChangeType: DeltaModified,
				Old:        oldVal,
				New:        newVal,
				Evidence:   "section value changed between existing rules and fresh scout",
			})
		}
	}

	// Detect ADDED: sections present in fresh but not in existing.
	for section, newVal := range fresh {
		if _, found := existing[section]; !found {
			deltas = append(deltas, Delta{
				Section:    section,
				ChangeType: DeltaAdded,
				New:        newVal,
				Evidence:   "section detected by fresh scout but absent from existing rules",
			})
		}
	}

	sort.Slice(deltas, func(i, j int) bool {
		if deltas[i].Section != deltas[j].Section {
			return deltas[i].Section < deltas[j].Section
		}
		return deltas[i].ChangeType < deltas[j].ChangeType
	})

	return deltas
}

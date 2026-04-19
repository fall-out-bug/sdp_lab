package rules

import "fmt"

// Diff represents a single change between two rule sets.
type Diff struct {
	Type     string // "added", "removed", "modified"
	Rule     Rule
	OldValue string // for modified
	NewValue string // for modified
}

// DiffRules compares two rule slices keyed by ID and returns the diffs.
// old is the baseline; new is the updated set.
func DiffRules(old, new []Rule) []Diff {
	oldMap := indexByID(old)
	newMap := indexByID(new)

	var diffs []Diff

	// Detect added and modified.
	for id, nr := range newMap {
		or, existed := oldMap[id]
		if !existed {
			diffs = append(diffs, Diff{Type: "added", Rule: nr})
			continue
		}
		if ruleChanged(or, nr) {
			diffs = append(diffs, Diff{
				Type:     "modified",
				Rule:     nr,
				OldValue: fmtRule(or),
				NewValue: fmtRule(nr),
			})
		}
	}

	// Detect removed.
	for id, or := range oldMap {
		if _, exists := newMap[id]; !exists {
			diffs = append(diffs, Diff{Type: "removed", Rule: or})
		}
	}

	return diffs
}

// indexByID builds a map from rule ID to rule.
func indexByID(rules []Rule) map[string]Rule {
	m := make(map[string]Rule, len(rules))
	for _, r := range rules {
		m[r.ID] = r
	}
	return m
}

// ruleChanged reports whether two rules differ in any field except ID.
func ruleChanged(a, b Rule) bool {
	return a.Title != b.Title ||
		a.Source != b.Source ||
		a.EvidenceRef != b.EvidenceRef ||
		a.Severity != b.Severity ||
		a.Description != b.Description
}

// fmtRule returns a human-readable summary of a rule for diff display.
func fmtRule(r Rule) string {
	return fmt.Sprintf("%s|%s|%s|%s", r.Title, r.Source, r.Severity, r.Description)
}

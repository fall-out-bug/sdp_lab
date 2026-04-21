package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffRules_Added(t *testing.T) {
	old := []Rule{
		{ID: "RULE-001", Title: "A", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
	}
	new := []Rule{
		{ID: "RULE-001", Title: "A", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
		{ID: "RULE-002", Title: "B", Source: SourceObservedFailure, Severity: SeverityWarning, Description: "d2"},
	}

	diffs := DiffRules(old, new)
	requireOneDiffOfType(t, diffs, "added")
}

func TestDiffRules_Removed(t *testing.T) {
	old := []Rule{
		{ID: "RULE-001", Title: "A", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
		{ID: "RULE-002", Title: "B", Source: SourceObservedFailure, Severity: SeverityWarning, Description: "d2"},
	}
	new := []Rule{
		{ID: "RULE-001", Title: "A", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
	}

	diffs := DiffRules(old, new)
	requireOneDiffOfType(t, diffs, "removed")
}

func TestDiffRules_Modified(t *testing.T) {
	old := []Rule{
		{ID: "RULE-001", Title: "Old Title", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
	}
	new := []Rule{
		{ID: "RULE-001", Title: "New Title", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
	}

	diffs := DiffRules(old, new)
	requireOneDiffOfType(t, diffs, "modified")

	d := firstDiffOfType(diffs, "modified")
	assert.Contains(t, d.OldValue, "Old Title")
	assert.Contains(t, d.NewValue, "New Title")
}

func TestDiffRules_NoChange(t *testing.T) {
	rules := []Rule{
		{ID: "RULE-001", Title: "A", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
	}
	diffs := DiffRules(rules, rules)
	assert.Empty(t, diffs)
}

func TestDiffRules_BothEmpty(t *testing.T) {
	diffs := DiffRules(nil, nil)
	assert.Empty(t, diffs)
}

func TestDiffRules_MultipleChanges(t *testing.T) {
	old := []Rule{
		{ID: "RULE-001", Title: "A", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
		{ID: "RULE-002", Title: "B", Source: SourceObservedFailure, Severity: SeverityWarning, Description: "d2"},
	}
	new := []Rule{
		{ID: "RULE-001", Title: "A-modified", Source: SourceObservedFailure, Severity: SeverityError, Description: "d1"},
		{ID: "RULE-003", Title: "C", Source: SourceHumanAnnotated, Severity: SeverityInfo, Description: "d3"},
	}

	diffs := DiffRules(old, new)
	assert.Len(t, diffs, 3) // RULE-001 modified, RULE-002 removed, RULE-003 added
}

func requireOneDiffOfType(t *testing.T, diffs []Diff, typ string) {
	t.Helper()
	count := 0
	for _, d := range diffs {
		if d.Type == typ {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 %s diff, got %d", typ, count)
	}
}

func firstDiffOfType(diffs []Diff, typ string) Diff {
	for _, d := range diffs {
		if d.Type == typ {
			return d
		}
	}
	return Diff{}
}

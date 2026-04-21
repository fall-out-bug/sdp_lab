package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator("/tmp/evidence")
	assert.NotNil(t, g)
	assert.Equal(t, "/tmp/evidence", g.evidenceDir)
}

func TestGenerate_NoEvidence(t *testing.T) {
	dir := t.TempDir()
	g := NewGenerator(dir)

	rules, err := g.Generate()
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestGenerate_WithFailures(t *testing.T) {
	dir := setupEvidenceDir(t,
		evidenceEntry{RunID: "run-1", Phase: "test", Verdict: "fail", Summary: "test X failed"},
		evidenceEntry{RunID: "run-1", Phase: "build", Verdict: "pass", Summary: "ok"},
		evidenceEntry{RunID: "run-2", Phase: "lint", Verdict: "error", Summary: "unused import"},
	)

	g := NewGenerator(dir)
	rules, err := g.Generate()
	require.NoError(t, err)
	require.Len(t, rules, 2) // "fail" + "error", "pass" excluded

	// Verify IDs follow RULE-NNN format.
	for _, r := range rules {
		assert.Regexp(t, `^RULE-\d{3}$`, r.ID)
		assert.Equal(t, SourceObservedFailure, r.Source)
		assert.NotEmpty(t, r.EvidenceRef)
	}

	// Sorted by ID.
	assert.True(t, sort.SliceIsSorted(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	}))
}

func TestGenerate_NeverSpeculative(t *testing.T) {
	dir := setupEvidenceDir(t,
		evidenceEntry{RunID: "run-1", Phase: "test", Verdict: "fail", Summary: "test X failed"},
	)

	g := NewGenerator(dir)
	rules, err := g.Generate()
	require.NoError(t, err)

	for _, r := range rules {
		assert.NotEqual(t, "speculative", r.Source,
			"rules must never have speculative source")
		assert.True(t, allowedSources[r.Source],
			"rule source must be in allowed set, got: %s", r.Source)
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	entries := []evidenceEntry{
		{RunID: "run-1", Phase: "test", Verdict: "fail", Summary: "test A failed"},
		{RunID: "run-2", Phase: "build", Verdict: "error", Summary: "compile error"},
		{RunID: "run-3", Phase: "test", Verdict: "fail", Summary: "test B failed"},
	}

	dir1 := setupEvidenceDir(t, entries...)
	dir2 := setupEvidenceDir(t, entries...)

	g1 := NewGenerator(dir1)
	g2 := NewGenerator(dir2)

	r1, err1 := g1.Generate()
	r2, err2 := g2.Generate()

	require.NoError(t, err1)
	require.NoError(t, err2)

	require.Equal(t, len(r1), len(r2), "same input should produce same count")
	for i := range r1 {
		assert.Equal(t, r1[i].ID, r2[i].ID, "ID mismatch at index %d", i)
		assert.Equal(t, r1[i].Title, r2[i].Title, "Title mismatch at index %d", i)
		assert.Equal(t, r1[i].Source, r2[i].Source, "Source mismatch at index %d", i)
	}
}

func TestRuleSourceValidation(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		isValid bool
	}{
		{
			name: "valid observed-failure",
			rule: Rule{
				ID: "RULE-001", Title: "t", Source: SourceObservedFailure,
				EvidenceRef: "ref", Severity: SeverityError, Description: "d",
			},
			isValid: true,
		},
		{
			name: "valid human-annotated",
			rule: Rule{
				ID: "RULE-002", Title: "t", Source: SourceHumanAnnotated,
				EvidenceRef: "ref", Severity: SeverityWarning, Description: "d",
			},
			isValid: true,
		},
		{
			name: "speculative source rejected",
			rule: Rule{
				ID: "RULE-003", Title: "t", Source: "speculative",
				EvidenceRef: "ref", Severity: SeverityInfo, Description: "d",
			},
			isValid: false,
		},
		{
			name: "empty source rejected",
			rule: Rule{
				ID: "RULE-004", Title: "t", Source: "",
				EvidenceRef: "ref", Severity: SeverityInfo, Description: "d",
			},
			isValid: false,
		},
		{
			name: "unknown source rejected",
			rule: Rule{
				ID: "RULE-005", Title: "t", Source: "ai-generated",
				EvidenceRef: "ref", Severity: SeverityInfo, Description: "d",
			},
			isValid: false,
		},
		{
			name: "empty ID rejected",
			rule: Rule{
				ID: "", Title: "t", Source: SourceObservedFailure,
				EvidenceRef: "ref", Severity: SeverityError, Description: "d",
			},
			isValid: false,
		},
		{
			name: "empty evidence_ref rejected",
			rule: Rule{
				ID: "RULE-006", Title: "t", Source: SourceObservedFailure,
				EvidenceRef: "", Severity: SeverityError, Description: "d",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.rule.Valid())
		})
	}
}

func TestGenerate_GroupsSimilarFailures(t *testing.T) {
	dir := setupEvidenceDir(t,
		evidenceEntry{RunID: "run-1", Phase: "test", Verdict: "fail", Summary: "test X failed"},
		evidenceEntry{RunID: "run-2", Phase: "test", Verdict: "fail", Summary: "test X failed"},
		evidenceEntry{RunID: "run-3", Phase: "test", Verdict: "fail", Summary: "test Y failed"},
	)

	g := NewGenerator(dir)
	rules, err := g.Generate()
	require.NoError(t, err)

	// Two distinct summaries => two rules.
	require.Len(t, rules, 2)

	// Each rule should reference multiple evidence files for grouped entries.
	for _, r := range rules {
		assert.Contains(t, r.EvidenceRef, SourceObservedFailure+":")
	}
}

func TestGenerate_GroupsByPhaseAndSummary(t *testing.T) {
	dir := setupEvidenceDir(t,
		evidenceEntry{RunID: "run-1", Phase: "build", Verdict: "fail", Summary: "same msg"},
		evidenceEntry{RunID: "run-2", Phase: "test", Verdict: "fail", Summary: "same msg"},
	)

	g := NewGenerator(dir)
	rules, err := g.Generate()
	require.NoError(t, err)

	// Different phases => separate rules even with same summary.
	require.Len(t, rules, 2)
}

// --- helpers ---

type evidenceEntry struct {
	RunID   string
	Phase   string
	Verdict string
	Summary string
}

func setupEvidenceDir(t *testing.T, entries ...evidenceEntry) string {
	t.Helper()
	dir := t.TempDir()
	for i, e := range entries {
		obj := map[string]string{
			"run_id":   e.RunID,
			"phase":    e.Phase,
			"verdict":  e.Verdict,
			"summary":  e.Summary,
		}
		data, err := json.Marshal(obj)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, fmt.Sprintf("entry-%03d.json", i+1)), data, 0o644)
		require.NoError(t, err)
	}
	return dir
}


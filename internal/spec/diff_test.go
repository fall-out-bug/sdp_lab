package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffSpecs_NoChange(t *testing.T) {
	r := baseReport()
	p1, p2 := writeSnapshot(t, "old", r), writeSnapshot(t, "new", r)
	diff, err := DiffSpecs(p1, p2)
	require.NoError(t, err)
	for _, ch := range [][]Change{diff.APIChanges, diff.RuleChanges, diff.InvChanges, diff.SLAChanges} {
		assert.Empty(t, ch)
	}
	assert.Equal(t, DiffSummary{}, diff.Summary)
}

func TestDiffSpecs_API(t *testing.T) {
	oldF := writeSnapshot(t, "old", baseReport())
	// Added
	r := copyReport(baseReport())
	r.APIContracts.HTTPEndpoints = append(r.APIContracts.HTTPEndpoints, Endpoint{Method: "POST", Path: "/api/new", Handler: "NewH"})
	diff, err := DiffSpecs(oldF, writeSnapshot(t, "na", r))
	require.NoError(t, err)
	require.Len(t, diff.APIChanges, 1)
	assert.Equal(t, "added", diff.APIChanges[0].Category)
	// Removed
	r2 := copyReport(baseReport())
	r2.APIContracts.HTTPEndpoints = append(r2.APIContracts.HTTPEndpoints, Endpoint{Method: "DELETE", Path: "/api/old", Handler: "OldH"})
	diff, err = DiffSpecs(writeSnapshot(t, "or", r2), oldF)
	require.NoError(t, err)
	assert.Equal(t, "removed", diff.APIChanges[0].Category)
	// Modified
	r3 := copyReport(baseReport())
	r3.APIContracts.HTTPEndpoints[0].Handler = "UpdatedH"
	diff, err = DiffSpecs(oldF, writeSnapshot(t, "nm", r3))
	require.NoError(t, err)
	assert.Equal(t, "modified", diff.APIChanges[0].Category)
	assert.Equal(t, "handler changed", diff.APIChanges[0].Detail)
}

func TestDiffSpecs_Rules(t *testing.T) {
	oldF := writeSnapshot(t, "old", baseReport())
	r := copyReport(baseReport())
	r.BusinessRules.Validations = append(r.BusinessRules.Validations, ValidationRule{Location: "n.go", Field: "email", Description: "req"})
	diff, err := DiffSpecs(oldF, writeSnapshot(t, "n1", r))
	require.NoError(t, err)
	assertHasChange(t, diff.RuleChanges, "added", "n.go#email")
	r2 := copyReport(baseReport())
	r2.BusinessRules.Validations[0].Description = "updated"
	diff, err = DiffSpecs(oldF, writeSnapshot(t, "n2", r2))
	require.NoError(t, err)
	assertHasChange(t, diff.RuleChanges, "modified", "handler.go#name")
}

func TestDiffSpecs_Invariants(t *testing.T) {
	oldF := writeSnapshot(t, "old", baseReport())
	r := copyReport(baseReport())
	r.Invariants.Database = append(r.Invariants.Database, DBInvariant{Table: "orders", Column: "status", Detail: "NOT NULL"})
	diff, err := DiffSpecs(oldF, writeSnapshot(t, "n1", r))
	require.NoError(t, err)
	assertHasChange(t, diff.InvChanges, "added", "orders.status")
	// DB modified
	r2 := copyReport(baseReport())
	r2.Invariants.Database[0].Detail = "BIGINT PK"
	diff, err = DiffSpecs(oldF, writeSnapshot(t, "n2", r2))
	require.NoError(t, err)
	assertHasChange(t, diff.InvChanges, "modified", "users.id")
	// Type modified
	rOld := copyReport(baseReport())
	rOld.Invariants.TypeSystem = append(rOld.Invariants.TypeSystem, TypeInvariant{Category: "ta", Location: "s.go", Detail: "old"})
	rNew := copyReport(baseReport())
	rNew.Invariants.TypeSystem = append(rNew.Invariants.TypeSystem, TypeInvariant{Category: "ta", Location: "s.go", Detail: "new"})
	diff, err = DiffSpecs(writeSnapshot(t, "to", rOld), writeSnapshot(t, "tn", rNew))
	require.NoError(t, err)
	assertHasChange(t, diff.InvChanges, "modified", "type:")
	// Type + Conc + Arch added
	r3 := copyReport(baseReport())
	r3.Invariants.TypeSystem = append(r3.Invariants.TypeSystem, TypeInvariant{Category: "ta", Location: "s.go", Detail: "T"})
	r3.Invariants.Concurrency = append(r3.Invariants.Concurrency, ConcInvariant{Category: "mu", Location: "l.go", Detail: "mu"})
	r3.Invariants.Architectural = append(r3.Invariants.Architectural, ArchInvariant{Category: "iface", Location: "a.go", Detail: "S"})
	diff, err = DiffSpecs(oldF, writeSnapshot(t, "n3", r3))
	require.NoError(t, err)
	assertHasChange(t, diff.InvChanges, "added", "type:")
	assertHasChange(t, diff.InvChanges, "added", "concurrency:")
	assertHasChange(t, diff.InvChanges, "added", "architectural:")
}

func TestDiffSpecs_SLA(t *testing.T) {
	oldF := writeSnapshot(t, "old", baseReport())
	r := copyReport(baseReport())
	r.SLAParameters.Timeouts[0].Value = "60s"
	diff, err := DiffSpecs(oldF, writeSnapshot(t, "n1", r))
	require.NoError(t, err)
	require.Len(t, diff.SLAChanges, 1)
	assert.Equal(t, "modified", diff.SLAChanges[0].Category)
	// Added
	r2 := copyReport(baseReport())
	r2.SLAParameters.Retries = append(r2.SLAParameters.Retries, SLAParam{Category: "retry", Component: "db", Value: "3"})
	diff, err = DiffSpecs(oldF, writeSnapshot(t, "n2", r2))
	require.NoError(t, err)
	assertHasChange(t, diff.SLAChanges, "added", "retry:")
	// Removed
	rOld := copyReport(baseReport())
	rOld.SLAParameters.RateLimits = append(rOld.SLAParameters.RateLimits, SLAParam{Category: "rate_limit", Component: "api", Value: "100"})
	diff, err = DiffSpecs(writeSnapshot(t, "ro", rOld), oldF)
	require.NoError(t, err)
	assertHasChange(t, diff.SLAChanges, "removed", "rate_limit:")
}

func TestDiffSpecs_Deterministic(t *testing.T) {
	oldF := writeSnapshot(t, "old", baseReport())
	r := copyReport(baseReport())
	r.APIContracts.HTTPEndpoints = append(r.APIContracts.HTTPEndpoints, Endpoint{Method: "POST", Path: "/new", Handler: "NewH"})
	newF := writeSnapshot(t, "new", r)
	d1, _ := DiffSpecs(oldF, newF)
	d2, _ := DiffSpecs(oldF, newF)
	assert.Equal(t, d1.APIChanges, d2.APIChanges)
	assert.Equal(t, d1.Summary, d2.Summary)
}

func TestDiffSpecs_SummaryCounts(t *testing.T) {
	r := copyReport(baseReport())
	r.APIContracts.HTTPEndpoints = append(r.APIContracts.HTTPEndpoints, Endpoint{Method: "POST", Path: "/extra", Handler: "Extra"})
	diff, err := DiffSpecs(writeSnapshot(t, "old", r), writeSnapshot(t, "new", baseReport()))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, diff.Summary.Removed, 1)
	assert.Equal(t, 0, diff.Summary.Added)
}

func TestDiffSpecs_Errors(t *testing.T) {
	_, err := DiffSpecs("no.json", "nope.json")
	assert.Error(t, err)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "b.json"), []byte("x"), 0o644)
	goodF := writeSnapshot(t, "g", baseReport())
	_, err = DiffSpecs(goodF, filepath.Join(dir, "b.json"))
	assert.Error(t, err)
}

func TestDiffSpecs_FixtureFiles(t *testing.T) {
	diff, err := DiffSpecs(filepath.Join("testdata", "spec_v1.json"), filepath.Join("testdata", "spec_v2.json"))
	require.NoError(t, err)
	assert.NotEmpty(t, diff.APIChanges)
	assert.True(t, diff.Summary.Added > 0 || diff.Summary.Removed > 0 || diff.Summary.Modified > 0)
}

func assertHasChange(t *testing.T, ch []Change, cat, prefix string) {
	t.Helper()
	for _, c := range ch {
		if c.Category == cat && len(c.Key) >= len(prefix) && c.Key[:len(prefix)] == prefix {
			return
		}
	}
	t.Fatalf("expected %s change with prefix %q in %+v", cat, prefix, ch)
}

func copyReport(r *SpecReport) *SpecReport {
	data, _ := json.Marshal(r)
	var c SpecReport
	json.Unmarshal(data, &c)
	return &c
}

func baseReport() *SpecReport {
	return &SpecReport{Version: "1.0.0", Repo: "test",
		APIContracts: APIContracts{HTTPEndpoints: []Endpoint{{Method: "GET", Path: "/api/users", Handler: "ListUsers"}}, Total: 1},
		BusinessRules: BusinessRules{Validations: []ValidationRule{{Location: "handler.go", Field: "name", Description: "req"}}, Total: 1},
		Invariants:    Invariants{Database: []DBInvariant{{Table: "users", Column: "id", Detail: "PK"}}, Total: 1},
		SLAParameters: SLAParameters{Timeouts: []SLAParam{{Category: "timeout", Component: "srv", Value: "30s"}}, Total: 1}}
}

func writeSnapshot(t *testing.T, name string, r *SpecReport) string {
	t.Helper()
	data, err := json.MarshalIndent(r, "", "  ")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), name+".json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

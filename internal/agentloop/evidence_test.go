package agentloop

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvidenceAccumulator_newHasEmptyState(t *testing.T) {
	ea := NewEvidenceAccumulator()
	snap := ea.Snapshot(RoleDiscover)
	assert.Empty(t, snap.Evidence)
	assert.Empty(t, snap.Claims)
	assert.Empty(t, snap.Quality)
}

func TestEvidenceAccumulator_onToolError_recordsNegativeEvidence(t *testing.T) {
	ea := NewEvidenceAccumulator()
	err := ea.OnToolResult(ToolResult{
		ID:   "tc1",
		Name: "bash",
		Err:  errors.New("exit status 1"),
	})
	require.NoError(t, err, "OnToolResult must not return error for tool failures")

	snap := ea.Snapshot(RoleDiscover)
	require.Len(t, snap.Evidence, 1)
	assert.Contains(t, snap.Evidence[0], "tool_error:bash:exit status 1",
		"tool errors must be recorded as negative evidence")
	// Quality must NOT be set for failed tool call
	assert.False(t, snap.Quality["test"])
}

func TestEvidenceAccumulator_onBashSuccess_setsQuality(t *testing.T) {
	ea := NewEvidenceAccumulator()
	err := ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "--- PASS: TestFoo (0.01s)\nok  \tsdp_dev/internal/agentloop\t0.123s",
	})
	require.NoError(t, err)

	snap := ea.Snapshot(RoleBuild)
	assert.True(t, snap.Quality["test"], "bash output containing PASS must set quality[test]=true")
}

func TestEvidenceAccumulator_onBashFailOutput_doesNotSetQuality(t *testing.T) {
	ea := NewEvidenceAccumulator()
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "--- FAIL: TestBar (0.05s)\nFAIL",
	}))

	snap := ea.Snapshot(RoleBuild)
	assert.False(t, snap.Quality["test"], "bash FAIL output must not set quality[test]")
}

func TestEvidenceAccumulator_onEditFile_recordsEvidence(t *testing.T) {
	ea := NewEvidenceAccumulator()
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "edit_file",
		Output: "edited: internal/agentloop/session.go",
	}))

	snap := ea.Snapshot(RoleBuild)
	require.Len(t, snap.Evidence, 1)
	assert.Equal(t, "file_modified:internal/agentloop/session.go", snap.Evidence[0])
}

func TestEvidenceAccumulator_onBdCreate_recordsEvidence(t *testing.T) {
	ea := NewEvidenceAccumulator()
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bd_create",
		Output: "card_created:PROJ-42",
	}))

	snap := ea.Snapshot(RolePlan)
	require.Len(t, snap.Evidence, 1)
	assert.Equal(t, "card_created:PROJ-42", snap.Evidence[0])
}

func TestEvidenceAccumulator_reset_clearsAll(t *testing.T) {
	ea := NewEvidenceAccumulator()

	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "PASS",
	}))
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:   "tc2",
		Name: "edit_file",
		Output: "edited: foo.go",
	}))

	snap := ea.Snapshot(RoleBuild)
	assert.NotEmpty(t, snap.Evidence)

	ea.Reset()

	snap2 := ea.Snapshot(RoleBuild)
	assert.Empty(t, snap2.Evidence, "Reset must clear evidence")
	assert.Empty(t, snap2.Quality, "Reset must clear quality")
	assert.Empty(t, snap2.Claims, "Reset must clear claims")
}

func TestEvidenceAccumulator_reset_allowsReuse(t *testing.T) {
	// After Reset, OnToolResult must still work (no nil map panic — Fix Q2).
	ea := NewEvidenceAccumulator()
	ea.Reset()

	err := ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "ok  \tsdp_dev\t0.01s",
	})
	require.NoError(t, err, "OnToolResult after Reset must not panic")
}

func TestEvidenceAccumulator_snapshot_concurrent(t *testing.T) {
	// Race detector test: concurrent OnToolResult + Snapshot must not data-race.
	ea := NewEvidenceAccumulator()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = ea.OnToolResult(ToolResult{ID: "tc", Name: "bash", Output: "PASS"})
		}()
		go func() {
			defer wg.Done()
			_ = ea.Snapshot(RoleBuild)
		}()
	}
	wg.Wait()
}

func TestEvidenceAccumulator_snapshot_returnsPhase(t *testing.T) {
	ea := NewEvidenceAccumulator()
	snap := ea.Snapshot(RoleReview)
	assert.Equal(t, RoleReview, snap.Phase)
}

func TestEvidenceAccumulator_toHarness_includesEvidence(t *testing.T) {
	ea := NewEvidenceAccumulator()
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "PASS",
	}))

	snap := ea.Snapshot(RoleBuild)
	hs := snap.toHarness()
	assert.Equal(t, "build", hs.Phase)
	assert.NotNil(t, hs.QualityResults)
	assert.True(t, hs.QualityResults["test"])
}

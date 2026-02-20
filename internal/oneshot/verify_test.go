package oneshot

import (
	"reflect"
	"testing"
)

func testManifest(t *testing.T) ExecutionManifest {
	t.Helper()
	manifest, err := BuildExecutionManifest(PlannerGraph{Nodes: []PlannerNode{
		{ID: "plan", Owner: "analyst", Artifacts: []string{"manifest-1"}},
		{ID: "build", Owner: "coder", DependsOn: []string{"plan"}, Artifacts: []string{"diff-1", "tests-1"}},
		{ID: "review", Owner: "reviewer", DependsOn: []string{"build"}, Artifacts: []string{"verdict-1"}},
	}})
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	return manifest
}

func TestVerifyRoleEvidenceComplete(t *testing.T) {
	manifest := testManifest(t)
	report := VerifyRoleEvidence(manifest, []RoleEvidence{
		{TaskID: "plan", Role: "analyst", Status: "ok", ArtifactIDs: []string{"manifest-1"}},
		{TaskID: "build", Role: "coder", Status: "ok", ArtifactIDs: []string{"diff-1", "tests-1"}},
		{TaskID: "review", Role: "reviewer", Status: "ok", ArtifactIDs: []string{"verdict-1"}, ConsumedArtifactIDs: []string{"diff-1"}},
	})

	if !report.OK {
		t.Fatalf("expected complete evidence report, got %+v", report)
	}
}

func TestVerifyRoleEvidenceDetectsGaps(t *testing.T) {
	manifest := testManifest(t)
	report := VerifyRoleEvidence(manifest, []RoleEvidence{
		{TaskID: "plan", Role: "coder", Status: "ok", ArtifactIDs: []string{"manifest-1"}},
		{TaskID: "build", Role: "coder", Status: "unknown", ArtifactIDs: []string{"diff-1"}},
	})

	if report.OK {
		t.Fatalf("expected failure report, got %+v", report)
	}
	if !reflect.DeepEqual(report.MissingTaskEvidence, []string{"review"}) {
		t.Fatalf("unexpected missing evidence: %#v", report.MissingTaskEvidence)
	}
	if len(report.RoleMismatches) != 1 {
		t.Fatalf("expected one role mismatch, got %#v", report.RoleMismatches)
	}
	if !reflect.DeepEqual(report.InvalidStatuses, []string{"build=unknown"}) {
		t.Fatalf("unexpected status failures: %#v", report.InvalidStatuses)
	}
}

func TestVerifyRoleEvidenceReviewerDependencyCoverage(t *testing.T) {
	manifest := testManifest(t)
	report := VerifyRoleEvidence(manifest, []RoleEvidence{
		{TaskID: "plan", Role: "analyst", Status: "ok", ArtifactIDs: []string{"manifest-1"}},
		{TaskID: "build", Role: "coder", Status: "ok", ArtifactIDs: []string{"diff-1", "tests-1"}},
		{TaskID: "review", Role: "reviewer", Status: "ok", ArtifactIDs: []string{"verdict-1"}, ConsumedArtifactIDs: []string{"manifest-1"}},
	})

	if report.OK {
		t.Fatalf("expected dependency coverage failure, got %+v", report)
	}
	if !reflect.DeepEqual(report.ReviewerDependencyGaps["review"], []string{"build"}) {
		t.Fatalf("unexpected reviewer dependency gap report: %#v", report.ReviewerDependencyGaps)
	}
}

func TestPlanFailureRecoveryIncludesDependents(t *testing.T) {
	manifest := testManifest(t)
	plan, err := PlanFailureRecovery(manifest, []string{"build"})
	if err != nil {
		t.Fatalf("plan recovery: %v", err)
	}
	if !reflect.DeepEqual(plan.RequeueTaskIDs, []string{"build", "review"}) {
		t.Fatalf("unexpected recovery scope: %#v", plan.RequeueTaskIDs)
	}
}

func TestPlanFailureRecoveryRejectsUnknownTask(t *testing.T) {
	manifest := testManifest(t)
	if _, err := PlanFailureRecovery(manifest, []string{"missing"}); err == nil {
		t.Fatal("expected unknown task error")
	}
}

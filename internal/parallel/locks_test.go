package parallel

import (
	"reflect"
	"testing"
)

func TestIdentifyHazardsDetectsDomains(t *testing.T) {
	paths := []string{
		"internal/policy/decision.go",
		".beads/issues/sdp_dev-2aq.19.1.md",
		".sdp/evidence/sdp_dev-2aq.19.1.json",
		"deploy/control-plane/deployment.yaml",
	}

	out := IdentifyHazards(paths)
	got := make([]string, 0, len(out))
	for _, hazard := range out {
		got = append(got, hazard.Key)
	}

	want := []string{
		"beads-state-race",
		"cluster-rollout-collision",
		"evidence-trace-interleave",
		"repo-tree-conflict",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected hazard keys: got=%v want=%v", got, want)
	}
}

func TestBuildLockRequestsStableAndScoped(t *testing.T) {
	paths := []string{
		"scripts/orchestrate_k8s_issue.sh",
		"scripts/apply_control_manifests.sh",
		"internal/oneshot/manifest.go",
		"internal/oneshot/manifest.go",
	}

	out := BuildLockRequests(paths, "autonomy/sdp_dev-2aq.19.1")

	want := []LockRequest{
		{Domain: DomainBranchRef, Scope: "autonomy/sdp_dev-2aq.19.1", Reason: "branch-update-serialization"},
		{Domain: DomainK8sControlPlane, Scope: "global", Reason: "cluster-rollout-collision"},
		{Domain: DomainRepoTree, Scope: "global", Reason: "repo-tree-conflict"},
	}

	if !reflect.DeepEqual(out, want) {
		t.Fatalf("unexpected lock requests: got=%#v want=%#v", out, want)
	}
}

func TestBuildLockRequestsWithoutSignals(t *testing.T) {
	out := BuildLockRequests([]string{"README.md"}, "")
	if len(out) != 0 {
		t.Fatalf("expected empty lock requests, got %#v", out)
	}
}

package parallel

import (
	"reflect"
	"testing"
)

func TestCanonicalizeLockRequestsUsesHierarchyAndScope(t *testing.T) {
	in := []LockRequest{
		{Domain: DomainBranchRef, Scope: "feature/x", Reason: "branch-update-serialization"},
		{Domain: DomainRepoTree, Scope: "global", Reason: "repo-tree-conflict"},
		{Domain: DomainBranchRef, Scope: "global", Reason: "fallback-serialize"},
		{Domain: DomainEvidenceStore, Scope: " ", Reason: "evidence-trace-interleave"},
	}

	out := CanonicalizeLockRequests(in)
	want := []LockRequest{
		{Domain: DomainEvidenceStore, Scope: "global", Reason: "evidence-trace-interleave"},
		{Domain: DomainRepoTree, Scope: "global", Reason: "repo-tree-conflict"},
		{Domain: DomainBranchRef, Scope: "global", Reason: "fallback-serialize"},
	}

	if !reflect.DeepEqual(out, want) {
		t.Fatalf("unexpected canonical locks: got=%#v want=%#v", out, want)
	}
}

func TestClassifyMergeQueue(t *testing.T) {
	tests := []struct {
		name string
		in   []LockRequest
		want MergeQueueClass
	}{
		{
			name: "no locks",
			in:   nil,
			want: MergeQueueConcurrentSafe,
		},
		{
			name: "branch scoped only",
			in: []LockRequest{
				{Domain: DomainBranchRef, Scope: "feature/a", Reason: "branch-update-serialization"},
			},
			want: MergeQueueSerialBranch,
		},
		{
			name: "branch global lock",
			in: []LockRequest{
				{Domain: DomainBranchRef, Scope: "global", Reason: "repo-tree-conflict"},
			},
			want: MergeQueueSerialGlobal,
		},
		{
			name: "non branch lock",
			in: []LockRequest{
				{Domain: DomainK8sControlPlane, Scope: "global", Reason: "cluster-rollout-collision"},
			},
			want: MergeQueueSerialGlobal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyMergeQueue(tc.in); got != tc.want {
				t.Fatalf("unexpected queue class: got=%s want=%s", got, tc.want)
			}
		})
	}
}

func TestEffectivePriorityAgesBounded(t *testing.T) {
	if got := EffectivePriority(3, 0); got != 3 {
		t.Fatalf("unexpected no-wait priority: got=%d want=3", got)
	}
	if got := EffectivePriority(3, 6); got != 1 {
		t.Fatalf("unexpected aged priority: got=%d want=1", got)
	}
	if got := EffectivePriority(2, 99); got != 0 {
		t.Fatalf("unexpected saturated priority: got=%d want=0", got)
	}
	if got := EffectivePriority(8, 0); got != 4 {
		t.Fatalf("unexpected clamped base priority: got=%d want=4", got)
	}
}

func TestBuildSchedulerPlanFromIntakeSignals(t *testing.T) {
	paths := []string{
		"internal/parallel/policy.go",
		".beads/issues/sdp_dev-2aq.19.2.md",
		".sdp/evidence/sdp_dev-2aq.19.2.json",
	}

	plan := BuildSchedulerPlan(paths, "autonomy/sdp_dev-2aq.19.2", 2, 4)
	wantLocks := []LockRequest{
		{Domain: DomainBeadsState, Scope: "global", Reason: "beads-state-race"},
		{Domain: DomainEvidenceStore, Scope: "global", Reason: "evidence-trace-interleave"},
		{Domain: DomainRepoTree, Scope: "global", Reason: "repo-tree-conflict"},
		{Domain: DomainBranchRef, Scope: "autonomy/sdp_dev-2aq.19.2", Reason: "branch-update-serialization"},
	}

	if !reflect.DeepEqual(plan.Locks, wantLocks) {
		t.Fatalf("unexpected plan locks: got=%#v want=%#v", plan.Locks, wantLocks)
	}
	if plan.Queue != MergeQueueSerialGlobal {
		t.Fatalf("unexpected queue class: got=%s want=%s", plan.Queue, MergeQueueSerialGlobal)
	}
	if plan.EffectivePriority != 1 {
		t.Fatalf("unexpected effective priority: got=%d want=1", plan.EffectivePriority)
	}
}

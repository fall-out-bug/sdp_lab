package parallel

import (
	"sort"
	"sync"
	"testing"
)

func TestInMemoryLockManagerDeterministicConflictsAndRelease(t *testing.T) {
	mgr := NewInMemoryLockManager()

	ok, conflicts := mgr.TryAcquire("build-a", []LockRequest{{Domain: DomainRepoTree, Scope: "global", Reason: "repo-tree-conflict"}})
	if !ok || len(conflicts) != 0 {
		t.Fatalf("expected first acquire success, ok=%v conflicts=%v", ok, conflicts)
	}

	ok, conflicts = mgr.TryAcquire("build-b", []LockRequest{{Domain: DomainRepoTree, Scope: "feature/x", Reason: "repo-tree-conflict"}})
	if ok {
		t.Fatalf("expected conflicting acquire to fail")
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected one conflict, got=%v", conflicts)
	}
	if conflicts[0].Domain != DomainRepoTree || conflicts[0].Scope != "global" || conflicts[0].Holder != "build-a" {
		t.Fatalf("unexpected conflict detail: %#v", conflicts[0])
	}

	mgr.Release("build-a")
	ok, conflicts = mgr.TryAcquire("build-b", []LockRequest{{Domain: DomainRepoTree, Scope: "feature/x", Reason: "repo-tree-conflict"}})
	if !ok || len(conflicts) != 0 {
		t.Fatalf("expected acquire success after release, ok=%v conflicts=%v", ok, conflicts)
	}
}

func TestDecideBuildAdmissionByQueueClass(t *testing.T) {
	plan := SchedulerPlan{Queue: MergeQueueSerialBranch}

	deniedNoBranch := DecideBuildAdmission(plan, MergeQueueSerialBranch, "", "", map[string]string{}, "build-1")
	if deniedNoBranch.Admit || deniedNoBranch.Reason != "serial-branch-requires-branch" {
		t.Fatalf("unexpected no-branch decision: %#v", deniedNoBranch)
	}

	allowed := DecideBuildAdmission(plan, MergeQueueSerialBranch, "feature/x", "", map[string]string{"feature/y": "other"}, "build-1")
	if !allowed.Admit || allowed.Reason != "branch-queue-free" {
		t.Fatalf("unexpected allowed decision: %#v", allowed)
	}

	deniedBusy := DecideBuildAdmission(plan, MergeQueueSerialBranch, "feature/x", "", map[string]string{"feature/x": "other"}, "build-1")
	if deniedBusy.Admit || deniedBusy.Reason != "branch-queue-busy" {
		t.Fatalf("unexpected busy branch decision: %#v", deniedBusy)
	}
}

func TestInMemoryBuildSchedulerQueueAndLockIntegration(t *testing.T) {
	scheduler := NewInMemoryBuildScheduler(nil)

	planA := SchedulerPlan{
		Locks: []LockRequest{{Domain: DomainBranchRef, Scope: "feature/a", Reason: "branch-update-serialization"}},
		Queue: MergeQueueSerialBranch,
	}
	planA2 := SchedulerPlan{
		Locks: []LockRequest{{Domain: DomainBranchRef, Scope: "feature/a", Reason: "branch-update-serialization"}},
		Queue: MergeQueueSerialBranch,
	}
	planB := SchedulerPlan{
		Locks: []LockRequest{{Domain: DomainBranchRef, Scope: "feature/b", Reason: "branch-update-serialization"}},
		Queue: MergeQueueSerialBranch,
	}

	if decision := scheduler.Admit("run-a", "feature/a", planA, planA.Queue); !decision.Admit {
		t.Fatalf("expected run-a admit, got %#v", decision)
	}
	if decision := scheduler.Admit("run-a2", "feature/a", planA2, planA2.Queue); decision.Admit || decision.Reason != "branch-queue-busy" {
		t.Fatalf("expected run-a2 branch queue denial, got %#v", decision)
	}
	if decision := scheduler.Admit("run-b", "feature/b", planB, planB.Queue); !decision.Admit {
		t.Fatalf("expected run-b admit on separate branch, got %#v", decision)
	}

	scheduler.Release("run-a")
	if decision := scheduler.Admit("run-a3", "feature/a", planA, planA.Queue); !decision.Admit {
		t.Fatalf("expected run-a3 admit after release, got %#v", decision)
	}
}

func TestAdmitBuildPlanUsesBuildSchedulerPlanAndMergeQueue(t *testing.T) {
	scheduler := NewInMemoryBuildScheduler(nil)

	first := scheduler.AdmitBuildPlan("global-1", []string{"internal/parallel/policy.go"}, "feature/a", 2, 0)
	if !first.Admit {
		t.Fatalf("expected first global serial build admitted, got %#v", first)
	}
	if first.Queue != MergeQueueSerialGlobal {
		t.Fatalf("unexpected queue class for first decision: got=%s", first.Queue)
	}

	second := scheduler.AdmitBuildPlan("global-2", []string{"internal/parallel/locks.go"}, "feature/b", 2, 0)
	if second.Admit || second.Reason != "global-queue-busy" {
		t.Fatalf("expected global queue denial, got %#v", second)
	}

	scheduler.Release("global-1")
	third := scheduler.AdmitBuildPlan("global-3", []string{"internal/parallel/locks.go"}, "feature/b", 2, 0)
	if !third.Admit {
		t.Fatalf("expected third build admitted after release, got %#v", third)
	}
}

func TestInMemoryLockManagerNoPartialAcquireOnConflict(t *testing.T) {
	mgr := NewInMemoryLockManager()

	ownerA := "run-a"
	ownerB := "run-b"
	ownerC := "run-c"

	ok, conflicts := mgr.TryAcquire(ownerA, []LockRequest{{Domain: DomainRepoTree, Scope: "global", Reason: "repo-tree-conflict"}})
	if !ok || len(conflicts) != 0 {
		t.Fatalf("expected ownerA to acquire repo-tree, ok=%v conflicts=%v", ok, conflicts)
	}

	ok, conflicts = mgr.TryAcquire(ownerB, []LockRequest{
		{Domain: DomainRepoTree, Scope: "global", Reason: "repo-tree-conflict"},
		{Domain: DomainK8sControlPlane, Scope: "global", Reason: "cluster-rollout-collision"},
	})
	if ok {
		t.Fatalf("expected ownerB to fail because repo-tree is held")
	}
	if len(conflicts) == 0 {
		t.Fatalf("expected conflicts for ownerB")
	}

	ok, conflicts = mgr.TryAcquire(ownerC, []LockRequest{{Domain: DomainK8sControlPlane, Scope: "global", Reason: "cluster-rollout-collision"}})
	if !ok || len(conflicts) != 0 {
		t.Fatalf("expected ownerC to acquire k8s lock (no partial acquire by ownerB), ok=%v conflicts=%v", ok, conflicts)
	}
}

func TestInMemoryBuildSchedulerRaceSimulationProgressAndFairRetries(t *testing.T) {
	scheduler := NewInMemoryBuildScheduler(nil)
	plan := SchedulerPlan{
		Locks: []LockRequest{{Domain: DomainRepoTree, Scope: "global", Reason: "repo-tree-conflict"}},
		Queue: MergeQueueSerialGlobal,
	}

	type result struct {
		id       string
		decision BuildSchedulerDecision
	}

	ids := []string{"run-01", "run-02", "run-03", "run-04", "run-05", "run-06"}
	start := make(chan struct{})
	results := make(chan result, len(ids))

	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- result{id: id, decision: scheduler.Admit(id, "feature/main", plan, plan.Queue)}
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	admitted := make([]string, 0, 1)
	denied := make([]string, 0, len(ids)-1)
	for outcome := range results {
		if outcome.decision.Admit {
			admitted = append(admitted, outcome.id)
			continue
		}
		if outcome.decision.Reason != "global-queue-busy" && outcome.decision.Reason != "lock-conflict" {
			t.Fatalf("unexpected denial reason for %s: %#v", outcome.id, outcome.decision)
		}
		denied = append(denied, outcome.id)
	}

	if len(admitted) != 1 {
		t.Fatalf("expected exactly one admitted contender in initial race, got=%d admitted=%v", len(admitted), admitted)
	}

	scheduler.Release(admitted[0])
	sort.Strings(denied)
	for _, id := range denied {
		next := scheduler.Admit(id, "feature/main", plan, plan.Queue)
		if !next.Admit {
			t.Fatalf("expected %s to admit on retry after release, got %#v", id, next)
		}
		scheduler.Release(id)
	}
}

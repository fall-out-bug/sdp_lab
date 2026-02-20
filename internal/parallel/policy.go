package parallel

import (
	"sort"
	"strings"
)

type MergeQueueClass string

const (
	MergeQueueConcurrentSafe MergeQueueClass = "concurrent-safe"
	MergeQueueSerialBranch   MergeQueueClass = "serial-branch"
	MergeQueueSerialGlobal   MergeQueueClass = "serial-global"
)

type SchedulerPlan struct {
	Locks             []LockRequest   `json:"locks"`
	Queue             MergeQueueClass `json:"queue"`
	EffectivePriority int             `json:"effective_priority"`
}

var lockHierarchyRank = map[LockDomain]int{
	DomainBeadsState:      10,
	DomainEvidenceStore:   20,
	DomainRepoTree:        30,
	DomainBranchRef:       40,
	DomainK8sControlPlane: 50,
}

func CanonicalizeLockRequests(requests []LockRequest) []LockRequest {
	byDomain := map[LockDomain]LockRequest{}
	for _, req := range requests {
		domain := req.Domain
		if domain == "" {
			continue
		}

		scope := normalizeScope(req.Scope)
		reason := strings.TrimSpace(req.Reason)
		if reason == "" {
			reason = "unspecified"
		}

		next := LockRequest{Domain: domain, Scope: scope, Reason: reason}
		current, exists := byDomain[domain]
		if !exists {
			byDomain[domain] = next
			continue
		}

		if current.Scope != "global" && next.Scope == "global" {
			byDomain[domain] = next
			continue
		}

		if current.Scope == next.Scope && next.Reason < current.Reason {
			byDomain[domain] = next
		}
	}

	out := make([]LockRequest, 0, len(byDomain))
	for _, req := range byDomain {
		out = append(out, req)
	}

	sort.Slice(out, func(i, j int) bool {
		left := out[i]
		right := out[j]
		leftRank := lockDomainRank(left.Domain)
		rightRank := lockDomainRank(right.Domain)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if left.Domain != right.Domain {
			return left.Domain < right.Domain
		}
		if left.Scope != right.Scope {
			return left.Scope < right.Scope
		}
		return left.Reason < right.Reason
	})

	return out
}

func lockDomainRank(domain LockDomain) int {
	rank, ok := lockHierarchyRank[domain]
	if ok {
		return rank
	}
	return 1000
}

func ClassifyMergeQueue(requests []LockRequest) MergeQueueClass {
	if len(requests) == 0 {
		return MergeQueueConcurrentSafe
	}

	onlyBranchScoped := true
	for _, req := range requests {
		scope := normalizeScope(req.Scope)
		if req.Domain != DomainBranchRef || scope == "global" {
			onlyBranchScoped = false
		}
		if req.Domain != DomainBranchRef {
			return MergeQueueSerialGlobal
		}
	}

	if onlyBranchScoped {
		return MergeQueueSerialBranch
	}
	return MergeQueueSerialGlobal
}

func EffectivePriority(basePriority int, waitCycles int) int {
	if basePriority < 0 {
		basePriority = 0
	}
	if basePriority > 4 {
		basePriority = 4
	}
	if waitCycles <= 0 {
		return basePriority
	}

	boost := waitCycles / 3
	if boost > basePriority {
		boost = basePriority
	}
	return basePriority - boost
}

func BuildSchedulerPlan(paths []string, branch string, basePriority int, waitCycles int) SchedulerPlan {
	raw := BuildLockRequests(paths, branch)
	locks := CanonicalizeLockRequests(raw)
	return SchedulerPlan{
		Locks:             locks,
		Queue:             ClassifyMergeQueue(locks),
		EffectivePriority: EffectivePriority(basePriority, waitCycles),
	}
}

func normalizeScope(scope string) string {
	trimmed := strings.TrimSpace(scope)
	if trimmed == "" {
		return "global"
	}
	return trimmed
}

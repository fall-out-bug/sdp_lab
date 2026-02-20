package parallel

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type LockConflict struct {
	Domain LockDomain `json:"domain"`
	Scope  string     `json:"scope"`
	Holder string     `json:"holder"`
}

type lockHandle struct {
	Domain LockDomain
	Scope  string
}

func (h lockHandle) String() string {
	return fmt.Sprintf("%s:%s", h.Domain, h.Scope)
}

type InMemoryLockManager struct {
	mu      sync.Mutex
	holders map[lockHandle]string
	owned   map[string]map[lockHandle]struct{}
}

func NewInMemoryLockManager() *InMemoryLockManager {
	return &InMemoryLockManager{
		holders: map[lockHandle]string{},
		owned:   map[string]map[lockHandle]struct{}{},
	}
}

func (m *InMemoryLockManager) TryAcquire(owner string, requests []LockRequest) (bool, []LockConflict) {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return false, []LockConflict{{Domain: "invalid", Scope: "global", Holder: "missing-owner"}}
	}

	canonical := CanonicalizeLockRequests(requests)
	m.mu.Lock()
	defer m.mu.Unlock()

	conflicts := make([]LockConflict, 0)
	for _, req := range canonical {
		scope := normalizeScope(req.Scope)
		h := lockHandle{Domain: req.Domain, Scope: scope}
		for held, holder := range m.holders {
			if holder == owner {
				continue
			}
			if held.Domain != h.Domain {
				continue
			}
			if held.Scope == "global" || h.Scope == "global" || held.Scope == h.Scope {
				conflicts = append(conflicts, LockConflict{Domain: held.Domain, Scope: held.Scope, Holder: holder})
			}
		}
	}

	if len(conflicts) > 0 {
		sort.Slice(conflicts, func(i, j int) bool {
			if conflicts[i].Domain != conflicts[j].Domain {
				return conflicts[i].Domain < conflicts[j].Domain
			}
			if conflicts[i].Scope != conflicts[j].Scope {
				return conflicts[i].Scope < conflicts[j].Scope
			}
			return conflicts[i].Holder < conflicts[j].Holder
		})
		return false, conflicts
	}

	if _, ok := m.owned[owner]; !ok {
		m.owned[owner] = map[lockHandle]struct{}{}
	}
	for _, req := range canonical {
		h := lockHandle{Domain: req.Domain, Scope: normalizeScope(req.Scope)}
		m.holders[h] = owner
		m.owned[owner][h] = struct{}{}
	}

	return true, nil
}

func (m *InMemoryLockManager) Release(owner string) {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	owned := m.owned[owner]
	for h := range owned {
		delete(m.holders, h)
	}
	delete(m.owned, owner)
}

type BuildSchedulerDecision struct {
	Admit     bool            `json:"admit"`
	Queue     MergeQueueClass `json:"queue"`
	Reason    string          `json:"reason"`
	Conflicts []LockConflict  `json:"conflicts,omitempty"`
}

func DecideBuildAdmission(plan SchedulerPlan, queue MergeQueueClass, branch string, activeGlobal string, activeByBranch map[string]string, runID string) BuildSchedulerDecision {
	resolvedQueue := queue
	if resolvedQueue == "" {
		resolvedQueue = plan.Queue
	}
	if resolvedQueue == "" {
		resolvedQueue = MergeQueueConcurrentSafe
	}

	branch = strings.TrimSpace(branch)
	runID = strings.TrimSpace(runID)
	activeGlobal = strings.TrimSpace(activeGlobal)

	switch resolvedQueue {
	case MergeQueueConcurrentSafe:
		return BuildSchedulerDecision{Admit: true, Queue: resolvedQueue, Reason: "concurrent-safe"}
	case MergeQueueSerialBranch:
		if branch == "" {
			return BuildSchedulerDecision{Admit: false, Queue: resolvedQueue, Reason: "serial-branch-requires-branch"}
		}
		owner := strings.TrimSpace(activeByBranch[branch])
		if owner != "" && owner != runID {
			return BuildSchedulerDecision{Admit: false, Queue: resolvedQueue, Reason: "branch-queue-busy"}
		}
		return BuildSchedulerDecision{Admit: true, Queue: resolvedQueue, Reason: "branch-queue-free"}
	case MergeQueueSerialGlobal:
		if activeGlobal != "" && activeGlobal != runID {
			return BuildSchedulerDecision{Admit: false, Queue: resolvedQueue, Reason: "global-queue-busy"}
		}
		return BuildSchedulerDecision{Admit: true, Queue: resolvedQueue, Reason: "global-queue-free"}
	default:
		return BuildSchedulerDecision{Admit: false, Queue: resolvedQueue, Reason: "unknown-merge-queue-class"}
	}
}

type InMemoryBuildScheduler struct {
	mu             sync.Mutex
	locks          *InMemoryLockManager
	activeGlobal   string
	activeByBranch map[string]string
	admitted       map[string]BuildSchedulerDecision
}

func NewInMemoryBuildScheduler(locks *InMemoryLockManager) *InMemoryBuildScheduler {
	if locks == nil {
		locks = NewInMemoryLockManager()
	}
	return &InMemoryBuildScheduler{
		locks:          locks,
		activeByBranch: map[string]string{},
		admitted:       map[string]BuildSchedulerDecision{},
	}
}

func (s *InMemoryBuildScheduler) Admit(id string, branch string, plan SchedulerPlan, queue MergeQueueClass) BuildSchedulerDecision {
	id = strings.TrimSpace(id)
	branch = strings.TrimSpace(branch)
	if id == "" {
		return BuildSchedulerDecision{Admit: false, Queue: queue, Reason: "missing-build-id"}
	}

	s.mu.Lock()
	decision := DecideBuildAdmission(plan, queue, branch, s.activeGlobal, s.activeByBranch, id)
	if !decision.Admit {
		s.mu.Unlock()
		return decision
	}
	s.mu.Unlock()

	ok, conflicts := s.locks.TryAcquire(id, plan.Locks)
	if !ok {
		decision.Admit = false
		decision.Reason = "lock-conflict"
		decision.Conflicts = conflicts
		return decision
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	decision = DecideBuildAdmission(plan, queue, branch, s.activeGlobal, s.activeByBranch, id)
	if !decision.Admit {
		s.locks.Release(id)
		return decision
	}

	if decision.Queue == MergeQueueSerialGlobal {
		s.activeGlobal = id
	}
	if decision.Queue == MergeQueueSerialBranch {
		s.activeByBranch[branch] = id
	}
	s.admitted[id] = decision
	return decision
}

func (s *InMemoryBuildScheduler) AdmitBuildPlan(id string, paths []string, branch string, basePriority int, waitCycles int) BuildSchedulerDecision {
	plan := BuildSchedulerPlan(paths, branch, basePriority, waitCycles)
	return s.Admit(id, branch, plan, plan.Queue)
}

func (s *InMemoryBuildScheduler) Release(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}

	s.mu.Lock()
	for branch, owner := range s.activeByBranch {
		if owner == id {
			delete(s.activeByBranch, branch)
		}
	}
	if s.activeGlobal == id {
		s.activeGlobal = ""
	}
	delete(s.admitted, id)
	s.mu.Unlock()

	s.locks.Release(id)
}

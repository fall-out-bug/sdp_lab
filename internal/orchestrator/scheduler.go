package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sdp_dev/internal/beads"
)

// Scheduler polls Beads for ready tasks and dispatches them with lock-domain semantics.
type Scheduler struct {
	adapter   *beads.Adapter
	workDir   string
	lockDir   string
	labels    []string
	limit     int
	mu        sync.Mutex
	activeSet map[string]struct{}
}

// NewScheduler returns a scheduler for the given working directory.
func NewScheduler(workDir string, labels []string, limit int) *Scheduler {
	if limit <= 0 {
		limit = 10
	}
	if len(labels) == 0 {
		labels = []string{"autonomy", "strict-evidence"}
	}
	return &Scheduler{
		adapter:   beads.NewAdapter(workDir),
		workDir:   workDir,
		lockDir:   filepath.Join(os.TempDir(), "sdp-orchestrate-locks"),
		labels:    labels,
		limit:     limit,
		activeSet: make(map[string]struct{}),
	}
}

// Ready returns issue IDs that are ready and not currently locked.
func (s *Scheduler) Ready() ([]string, error) {
	issues, err := s.adapter.Ready(s.labels, s.limit)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(issues))
	for _, iss := range issues {
		if _, ok := s.activeSet[iss.ID]; !ok {
			out = append(out, iss.ID)
		}
	}
	return out, nil
}

// TryLock attempts to acquire a lock for the issue. Returns true if acquired.
func (s *Scheduler) TryLock(issueID string) (bool, error) {
	lockPath := filepath.Join(s.lockDir, "lock-"+issueID)
	if err := os.MkdirAll(s.lockDir, 0o755); err != nil {
		return false, err
	}
	// Use mkdir as atomic lock (same as orchestrate script)
	err := os.Mkdir(lockPath, 0o700)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	s.mu.Lock()
	s.activeSet[issueID] = struct{}{}
	s.mu.Unlock()
	return true, nil
}

// Unlock releases the lock for the issue.
func (s *Scheduler) Unlock(issueID string) {
	lockPath := filepath.Join(s.lockDir, "lock-"+issueID)
	_ = os.Remove(lockPath)
	s.mu.Lock()
	delete(s.activeSet, issueID)
	s.mu.Unlock()
}

// PickOne returns the first ready issue that can be locked, or empty string.
func (s *Scheduler) PickOne() (string, error) {
	ready, err := s.Ready()
	if err != nil {
		return "", err
	}
	for _, id := range ready {
		ok, err := s.TryLock(id)
		if err != nil {
			return "", err
		}
		if ok {
			return id, nil
		}
	}
	return "", nil
}

// Adapter returns the beads adapter for direct use.
func (s *Scheduler) Adapter() *beads.Adapter {
	return s.adapter
}

// WorkDir returns the working directory.
func (s *Scheduler) WorkDir() string {
	return s.workDir
}

// RunID generates a run ID for the issue.
func RunID(issueID string) string {
	return fmt.Sprintf("orchestrate-%s-%s", issueID, time.Now().UTC().Format("20060102T150405Z"))
}

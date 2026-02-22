package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"sdp_dev/internal/safeid"
)

var _ RunLock = (*RunLockManager)(nil)

// RunLockManager provides idempotent issue-run locking and duplicate suppression.
type RunLockManager struct {
	lockDir string
	mu      sync.Mutex
	active  map[string]string // issueID -> runID
}

// NewRunLockManager returns a lock manager for the given directory.
func NewRunLockManager(lockDir string) *RunLockManager {
	if lockDir == "" {
		lockDir = os.TempDir()
	}
	return &RunLockManager{
		lockDir: lockDir,
		active:  make(map[string]string),
	}
}

// TryAcquire attempts to acquire a lock for the issue. Returns (runID, true) if acquired.
func (m *RunLockManager) TryAcquire(issueID, runID string) (string, bool, error) {
	if err := safeid.ValidateIssueID(issueID); err != nil {
		return "", false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.active[issueID]; ok {
		return existing, false, nil
	}
	path := filepath.Join(m.lockDir, "sdp-adapter-"+issueID+".lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", false, err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		if os.IsExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	m.active[issueID] = runID
	return runID, true, nil
}

// Release releases the lock for the issue.
func (m *RunLockManager) Release(issueID string) error {
	if err := safeid.ValidateIssueID(issueID); err != nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, issueID)
	path := filepath.Join(m.lockDir, "sdp-adapter-"+issueID+".lock")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("release lock: %w", err)
	}
	return nil
}

// IsLocked returns true if the issue has an active lock.
func (m *RunLockManager) IsLocked(issueID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.active[issueID]
	return ok
}

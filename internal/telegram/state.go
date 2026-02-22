package telegram

import (
	"sync"
)

// UserState holds per-user conversation state.
type UserState struct {
	ActiveDiscussID string
	PendingApprove  string
}

// StateStore holds user states in memory.
type StateStore struct {
	mu     sync.RWMutex
	states map[int64]*UserState
}

// NewStateStore returns a new state store.
func NewStateStore() *StateStore {
	return &StateStore{states: make(map[int64]*UserState)}
}

// Get returns the state for a user.
func (s *StateStore) Get(chatID int64) *UserState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.states[chatID]
	if st == nil {
		return &UserState{}
	}
	return st
}

// Set updates the state for a user.
func (s *StateStore) Set(chatID int64, st *UserState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st == nil {
		delete(s.states, chatID)
		return
	}
	s.states[chatID] = st
}

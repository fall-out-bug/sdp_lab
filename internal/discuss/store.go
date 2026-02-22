package discuss

import (
	"fmt"
	"sync"
	"time"
)

// Store provides in-memory session storage for discussion sessions.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	seq      int
}

// NewStore returns a new in-memory store.
func NewStore() *Store {
	return &Store{sessions: make(map[string]*Session)}
}

// Create creates a new session and returns its ID.
func (s *Store) Create(req DiscussRequest) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("discuss-%d-%d", time.Now().Unix(), s.seq)
	now := time.Now().UTC()
	sess := &Session{
		ID:          id,
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Source:      req.Source,
		UserID:     req.UserID,
		Phase:      PhaseCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if sess.ProjectID == "" {
		sess.ProjectID = "default"
	}
	s.sessions[id] = sess
	return sess, nil
}

// Get returns a session by ID.
func (s *Store) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// Update updates a session.
func (s *Store) Update(sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[sess.ID]; !ok {
		return fmt.Errorf("session not found: %s", sess.ID)
	}
	sess.UpdatedAt = time.Now().UTC()
	s.sessions[sess.ID] = sess
	return nil
}

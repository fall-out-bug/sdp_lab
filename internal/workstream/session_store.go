// Package workstream provides workstream session storage for ephemeral items.
package workstream

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Wisp is an ephemeral work item that exists only for the current session.
type Wisp struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	
	// Title is the wisp title.
	Title string `json:"title"`
	
	// Description is the wisp description.
	Description string `json:"description,omitempty"`
	
	// Type is the wisp type (task, bug, etc.).
	Type string `json:"type"`
	
	// Priority is the priority (1-3).
	Priority int `json:"priority"`
	
	// Status is the current status.
	Status string `json:"status"`
	
	// Labels are tags for categorization.
	Labels []string `json:"labels,omitempty"`
	
	// SourceSession is the session that created this wisp.
	SourceSession string `json:"source_session"`
	
	// CreatedAt is when the wisp was created.
	CreatedAt time.Time `json:"created_at"`
	
	// ExpiresAt is when the wisp expires.
	ExpiresAt time.Time `json:"expires_at"`
	
	// Metadata stores additional data.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SessionStore manages ephemeral work items for a session.
type SessionStore struct {
	mu       sync.RWMutex
	basePath string
	ttl      time.Duration
}

// SessionStoreConfig configures the session store.
type SessionStoreConfig struct {
	// BasePath is the .sdp/session directory.
	BasePath string
	
	// TTL is the default wisp lifetime.
	TTL time.Duration
}

// DefaultWispTTL is the default wisp lifetime (24 hours).
const DefaultWispTTL = 24 * time.Hour

// NewSessionStore creates a new session store.
func NewSessionStore(cfg SessionStoreConfig) (*SessionStore, error) {
	basePath := cfg.BasePath
	if basePath == "" {
		// Default to .sdp/session
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get cwd: %w", err)
		}
		basePath = filepath.Join(cwd, ".sdp", "session")
	}
	
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = DefaultWispTTL
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}
	
	return &SessionStore{
		basePath: basePath,
		ttl:      ttl,
	}, nil
}

// CreateWisp creates a new ephemeral work item.
func (s *SessionStore) CreateWisp(w Wisp) (*Wisp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Generate ID if not set
	if w.ID == "" {
		w.ID = s.generateWispID(w.Title, time.Now())
	}
	
	// Set defaults
	if w.Status == "" {
		w.Status = "open"
	}
	if w.Type == "" {
		w.Type = "task"
	}
	if w.Priority == 0 {
		w.Priority = 2
	}
	if w.CreatedAt.IsZero() {
		w.CreatedAt = time.Now()
	}
	if w.ExpiresAt.IsZero() {
		w.ExpiresAt = w.CreatedAt.Add(s.ttl)
	}
	
	// Write to file
	wispPath := filepath.Join(s.basePath, "wisps", w.ID+".json")
	if err := os.MkdirAll(filepath.Dir(wispPath), 0755); err != nil {
		return nil, fmt.Errorf("create wisps dir: %w", err)
	}
	
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal wisp: %w", err)
	}
	
	if err := os.WriteFile(wispPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write wisp: %w", err)
	}
	
	return &w, nil
}

// GetWisp retrieves a wisp by ID.
func (s *SessionStore) GetWisp(id string) (*Wisp, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	wispPath := filepath.Join(s.basePath, "wisps", id+".json")
	data, err := os.ReadFile(wispPath)
	if err != nil {
		return nil, fmt.Errorf("read wisp: %w", err)
	}
	
	var w Wisp
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("unmarshal wisp: %w", err)
	}
	
	return &w, nil
}

// UpdateWispStatus updates a wisp's status.
func (s *SessionStore) UpdateWispStatus(id, status string) error {
	w, err := s.GetWisp(id)
	if err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	w.Status = status
	
	wispPath := filepath.Join(s.basePath, "wisps", id+".json")
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal wisp: %w", err)
	}
	
	return os.WriteFile(wispPath, data, 0644)
}

// ListWisps lists all non-expired wisps.
func (s *SessionStore) ListWisps() ([]*Wisp, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	wispsDir := filepath.Join(s.basePath, "wisps")
	entries, err := os.ReadDir(wispsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read wisps dir: %w", err)
	}
	
	var wisps []*Wisp
	now := time.Now()
	
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		
		w, err := s.GetWisp(filepath.Base(entry.Name()[:len(entry.Name())-5]))
		if err != nil {
			continue
		}
		
		// Skip expired wisps
		if w.ExpiresAt.Before(now) {
			continue
		}
		
		wisps = append(wisps, w)
	}
	
	return wisps, nil
}

// ExpireWisps removes all expired wisps.
func (s *SessionStore) ExpireWisps() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	wispsDir := filepath.Join(s.basePath, "wisps")
	entries, err := os.ReadDir(wispsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read wisps dir: %w", err)
	}
	
	now := time.Now()
	expired := 0
	
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		
		sid := entry.Name()[:len(entry.Name())-5]
		_ = sid // ID extracted from filename
		wispPath := filepath.Join(wispsDir, entry.Name())
		data, err := os.ReadFile(wispPath)
		if err != nil {
			continue
		}
		
		var w Wisp
		if err := json.Unmarshal(data, &w); err != nil {
			continue
		}
		
		if w.ExpiresAt.Before(now) {
			os.Remove(wispPath)
			expired++
		}
	}
	
	return expired, nil
}

// ClearSession removes all session data.
func (s *SessionStore) ClearSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	wispsDir := filepath.Join(s.basePath, "wisps")
	return os.RemoveAll(wispsDir)
}

// Stats returns session store statistics.
func (s *SessionStore) Stats() (*SessionStats, error) {
	wisps, err := s.ListWisps()
	if err != nil {
		return nil, err
	}
	
	stats := &SessionStats{
		ActiveWisps: len(wisps),
		ByStatus:    make(map[string]int),
		ByType:      make(map[string]int),
	}
	
	for _, w := range wisps {
		stats.ByStatus[w.Status]++
		stats.ByType[w.Type]++
	}
	
	return stats, nil
}

// SessionStats contains session statistics.
type SessionStats struct {
	ActiveWisps int            `json:"active_wisps"`
	ByStatus    map[string]int `json:"by_status"`
	ByType      map[string]int `json:"by_type"`
}

// generateWispID generates a unique wisp ID.
func (s *SessionStore) generateWispID(title string, t time.Time) string {
	h := sha256.New()
	h.Write([]byte(title))
	h.Write([]byte(t.String()))
	hash := hex.EncodeToString(h.Sum(nil))
	if len(hash) > 8 {
		return "wisp-" + hash[:8]
	}
	return "wisp-" + hash
}

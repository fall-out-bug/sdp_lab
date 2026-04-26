package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	ErrStoreClosed    = errors.New("store is closed")
	ErrWriteFailed    = errors.New("failed to write to store")
	ErrReadFailed     = errors.New("failed to read from store")
	ErrInvalidJSON    = errors.New("invalid JSON in entry")
	ErrSecretDetected = errors.New("secret detected in content")
)

// Store manages the append-only JSONL memory file.
type Store struct {
	mu       sync.Mutex
	file     *os.File
	path     string
	isClosed bool
}

// NewStore creates or opens a memory store at the given directory.
// The store file is named "memory.jsonl".
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	path := fmt.Sprintf("%s/memory.jsonl", dir)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open store file: %w", err)
	}

	return &Store{
		file:     file,
		path:     path,
		isClosed: false,
	}, nil
}

// Append adds a new entry to the store.
func (s *Store) Append(entry MemoryEntry) error {
	// Validate entry first
	if err := ValidateEntry(&entry); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Scan for secrets
	if found, pattern := ScanForSecrets(entry.Content); found {
		return fmt.Errorf("%w: detected %s", ErrSecretDetected, pattern)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed {
		return ErrStoreClosed
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	// Append newline and write
	data = append(data, '\n')
	if _, err := s.file.Write(data); err != nil {
		return fmt.Errorf("%w: %v", ErrWriteFailed, err)
	}

	return nil
}

// ReadAll reads all entries from the store file.
func (s *Store) ReadAll() ([]MemoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed {
		return nil, ErrStoreClosed
	}

	// Sync before reading
	if err := s.file.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync file: %w", err)
	}

	// Close write handle and open for reading
	if err := s.file.Close(); err != nil {
		return nil, fmt.Errorf("failed to close write handle: %w", err)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	// Reopen for appending
	s.file, err = os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen store file: %w", err)
	}

	// Parse JSONL
	var entries []MemoryEntry
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry MemoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Close closes the store file.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed {
		return nil
	}

	s.isClosed = true
	return s.file.Close()
}

// extractYAMLField extracts a field value from YAML frontmatter in content.
// Content format: "---\nkey: value\n---\nrest of content"
func extractYAMLField(content, field string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of frontmatter
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				// End of frontmatter
				break
			}
		}

		if !inFrontmatter {
			continue
		}

		// Check for field
		if strings.HasPrefix(trimmed, field+":") {
			// Extract value after ":"
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

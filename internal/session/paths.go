package session

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Paths provides canonical path helpers for session-related directories and files.
type Paths struct {
	ProjectRoot string
}

// validSessionIDRegex matches alphanumeric, hyphen, underscore session IDs
var validSessionIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateSessionID checks that a session ID is safe for use in file paths.
func validateSessionID(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if strings.Contains(sessionID, "..") {
		return fmt.Errorf("session ID %q contains invalid path traversal sequence", sessionID)
	}
	if strings.ContainsAny(sessionID, "/\\") {
		return fmt.Errorf("session ID %q contains invalid path separator", sessionID)
	}
	if !validSessionIDRegex.MatchString(sessionID) {
		return fmt.Errorf("session ID %q contains invalid characters", sessionID)
	}
	return nil
}

// NewPaths creates a Paths helper for the given project root.
func NewPaths(projectRoot string) *Paths {
	return &Paths{ProjectRoot: projectRoot}
}

// LogDir returns the session log directory (.sdp/log).
func (p *Paths) LogDir() string {
	return filepath.Join(p.ProjectRoot, defaultLogDir)
}

// CacheDir returns the cache directory (.sdp/cache).
func (p *Paths) CacheDir() string {
	return filepath.Join(p.ProjectRoot, ".sdp", "cache")
}

func (p *Paths) MemDir() string {
	return filepath.Join(p.ProjectRoot, ".sdp", "mem")
}

// SessionLog returns the path to a session log file.
// Returns an error if sessionID contains path traversal characters.
func (p *Paths) SessionLog(sessionID string) (string, error) {
	if err := validateSessionID(sessionID); err != nil {
		return "", err
	}
	return filepath.Join(p.LogDir(), "session-"+sessionID+".jsonl"), nil
}

// GuardScopeFile returns the path to the guard scope file.
func (p *Paths) GuardScopeFile() string {
	return filepath.Join(p.ProjectRoot, ".sdp", "guard-scope.json")
}

// EnsureLogDir creates the log directory if it doesn't exist.
func (p *Paths) EnsureLogDir() error {
	return os.MkdirAll(p.LogDir(), 0o755)
}

// EnsureCacheDir creates the cache directory if it doesn't exist.
func (p *Paths) EnsureCacheDir() error {
	return os.MkdirAll(p.CacheDir(), 0o755)
}

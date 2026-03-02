package session

import (
	"fmt"
"os"
"path/filepath"
)

// Paths provides canonical path helpers for session-related directories and files.
type Paths struct {
	ProjectRoot string
}

// NewPaths creates a Paths helper for the given project root.
func NewPaths(projectRoot string) *Paths {
	return &Paths{ProjectRoot: projectRoot}
}

// LogDir returns the session log directory (.sdp/log).
func (p *Paths) LogDir() string {
	return filepath.Join(p.ProjectRoot, DefaultLogDir)
}

// CacheDir returns the cache directory (.sdp/cache).
func (p *Paths) CacheDir() string {
	return filepath.Join(p.ProjectRoot, ".sdp", "cache")
}

// SessionLog returns the path to a session log file.
func (p *Paths) SessionLog(sessionID string) string {
	return filepath.Join(p.LogDir(), "session-"+sessionID+".jsonl")
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

// Global instance for convenience
var defaultPaths *Paths

// InitPaths initializes the default paths helper.
func InitPaths(projectRoot string) {
	defaultPaths = NewPaths(projectRoot)
}

// GetPaths returns the default paths helper.
// Must call InitPaths first. Returns ErrNotInitialized if InitPaths was not called.
func GetPaths() (*Paths, error) {
	if defaultPaths == nil {
		return nil, ErrNotInitialized
	}
	return defaultPaths, nil
}

// MustGetPaths returns the default paths helper or panics if not initialized.
// Use this only when initialization is guaranteed.
func MustGetPaths() *Paths {
	if defaultPaths == nil {
		panic("session: InitPaths not called")
	}
	return defaultPaths
}

// ErrNotInitialized is returned when GetPaths is called before InitPaths.
var ErrNotInitialized = fmt.Errorf("session: InitPaths not called")

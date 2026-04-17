// Package snapshot provides golden-file snapshot testing for CLI output validation.
//
// Snapshots are stored as plain text files under a .snapshots/ directory named
// <name>.snap.  Set UPDATE_SNAPSHOTS=1 to write/overwrite snapshots instead of
// comparing.
package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Snapshotter compares and stores golden-file snapshots for CLI output.
type Snapshotter struct {
	// SnapDir is the directory where snapshot files are stored (default: .snapshots).
	SnapDir string
}

// New creates a Snapshotter rooted at snapDir. If snapDir is empty it defaults
// to ".snapshots".
func New(snapDir string) *Snapshotter {
	if snapDir == "" {
		snapDir = ".snapshots"
	}
	return &Snapshotter{SnapDir: snapDir}
}

// ShouldUpdate returns true if UPDATE_SNAPSHOTS=1 is set in the environment.
func (s *Snapshotter) ShouldUpdate() bool {
	return os.Getenv("UPDATE_SNAPSHOTS") == "1"
}

// Compare validates that actual matches the stored snapshot for name.
// If ShouldUpdate() is true it writes the snapshot instead.
// Returns nil on match, or an error with a diff on mismatch.
func (s *Snapshotter) Compare(name, actual string) error {
	if s.ShouldUpdate() {
		return s.Update(name, actual)
	}
	path := s.snapshotPath(name)
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("snapshot %q not found (run with UPDATE_SNAPSHOTS=1 to create): %w", name, err)
	}
	expected := string(raw)
	if expected == actual {
		return nil
	}
	return fmt.Errorf("snapshot mismatch for %q:\n--- expected\n+++ actual\n%s",
		name, diffLines(expected, actual))
}

// Update writes (or overwrites) the snapshot file for name with actual content.
func (s *Snapshotter) Update(name, actual string) error {
	path := s.snapshotPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}
	return os.WriteFile(path, []byte(actual), 0o644)
}

func (s *Snapshotter) snapshotPath(name string) string {
	return filepath.Join(s.SnapDir, name+".snap")
}

// diffLines returns a simple line-level diff between two strings.
func diffLines(a, b string) string {
	al := splitLines(a)
	bl := splitLines(b)
	n := len(al)
	if len(bl) > n {
		n = len(bl)
	}
	var sb strings.Builder
	for i := 0; i < n; i++ {
		la, lb := "", ""
		if i < len(al) {
			la = al[i]
		}
		if i < len(bl) {
			lb = bl[i]
		}
		if la != lb {
			fmt.Fprintf(&sb, "- %s\n+ %s\n", la, lb)
		}
	}
	return sb.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

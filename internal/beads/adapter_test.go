package beads

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdapter_Ready(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Walk up to find repo root (contains .beads)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".beads")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("no .beads found, skipping")
			return
		}
		dir = parent
	}

	a := NewAdapter(dir)
	issues, err := a.Ready([]string{"autonomy"}, 5)
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	_ = issues // may be empty
}

func TestAdapter_Show(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".beads")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("no .beads found, skipping")
			return
		}
		dir = parent
	}

	a := NewAdapter(dir)
	// Use an issue that likely exists
	issue, err := a.Show("sdp_dev-cgk")
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if issue.ID == "" {
		t.Error("expected non-empty ID")
	}
}

package beads

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func skipIfBdNotInPath(t *testing.T) {
	if _, err := exec.LookPath("bd"); err != nil {
		t.Skipf("bd not in PATH, skipping integration test: %v", err)
	}
}

func TestParseIssueList(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		wantLen int
		wantErr bool
	}{
		{"empty", []byte(""), 0, false},
		{"whitespace", []byte("   \n"), 0, false},
		{"array", []byte(`[{"id":"x","title":"t"}]`), 1, false},
		{"object", []byte(`{"id":"y","title":"t2"}`), 1, false},
		{"prefixed", []byte("warn: ok\n[{\"id\":\"z\"}]"), 1, false},
		{"no json", []byte("no json here"), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIssueList(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIssueList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("parseIssueList() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestAdapter_Ready(t *testing.T) {
	skipIfBdNotInPath(t)
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

func TestAdapter_Closed(t *testing.T) {
	skipIfBdNotInPath(t)
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
	issues, err := a.Closed(nil, 5)
	if err != nil {
		t.Fatalf("Closed: %v", err)
	}
	_ = issues // may be empty
}

func TestAdapter_Show(t *testing.T) {
	skipIfBdNotInPath(t)
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

func TestAdapter_DepsClosed(t *testing.T) {
	skipIfBdNotInPath(t)
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
	// Issue with no deps should return true
	closed, err := a.DepsClosed("sdp_dev-5l9")
	if err != nil {
		t.Fatalf("DepsClosed: %v", err)
	}
	if !closed {
		t.Error("DepsClosed expected true for issue with no deps")
	}
}

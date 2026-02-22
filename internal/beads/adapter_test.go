package beads

import (
	"os"
	"path/filepath"
	"testing"
)

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

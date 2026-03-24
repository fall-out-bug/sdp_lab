package control

import (
	"testing"
)

func TestOpenWithMode_FileMode(t *testing.T) {
	// Use the actual sdp_lab project root
	store, err := OpenWithMode("/home/fall_out_bug/projects/vibe_coding/sdp_lab", RepoModeFile, "")
	if err != nil {
		t.Fatalf("OpenWithMode file: %v", err)
	}
	if store.RepoMode != RepoModeFile {
		t.Errorf("mode: got %s, want %s", store.RepoMode, RepoModeFile)
	}
	if store.BeadsRepo() != nil {
		t.Error("expected nil beads repo in file mode")
	}
	if store.DualRepo() != nil {
		t.Error("expected nil dual repo in file mode")
	}
}

func TestOpenWithMode_BeadsMode(t *testing.T) {
	store, err := OpenWithMode("/home/fall_out_bug/projects/vibe_coding/sdp_lab", RepoModeBeads, "")
	if err != nil {
		t.Fatalf("OpenWithMode beads: %v", err)
	}
	if store.RepoMode != RepoModeBeads {
		t.Errorf("mode: got %s, want %s", store.RepoMode, RepoModeBeads)
	}
	if store.BeadsRepo() == nil {
		t.Error("expected non-nil beads repo in beads mode")
	}
}

func TestOpenWithMode_DualMode(t *testing.T) {
	store, err := OpenWithMode("/home/fall_out_bug/projects/vibe_coding/sdp_lab", RepoModeDual, "")
	if err != nil {
		t.Fatalf("OpenWithMode dual: %v", err)
	}
	if store.RepoMode != RepoModeDual {
		t.Errorf("mode: got %s, want %s", store.RepoMode, RepoModeDual)
	}
	if store.BeadsRepo() == nil {
		t.Error("expected non-nil beads repo in dual mode")
	}
	if store.DualRepo() == nil {
		t.Error("expected non-nil dual repo in dual mode")
	}
}

func TestOpen_BackwardsCompatible(t *testing.T) {
	store, err := Open("/home/fall_out_bug/projects/vibe_coding/sdp_lab")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if store.RepoMode != RepoModeFile {
		t.Errorf("Open() should default to file mode, got %s", store.RepoMode)
	}
}

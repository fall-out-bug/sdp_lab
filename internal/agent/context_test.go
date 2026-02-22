package agent

import (
	"path/filepath"
	"testing"

	"sdp_dev/internal/bus"
)

func TestNewContext(t *testing.T) {
	dir := t.TempDir()
	ctx, err := NewContext(Config{
		WorkDir:   dir,
		ProjectID: "p1",
		RunID:     "r1",
		AgentID:   "a1",
		Role:      "coder",
	})
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	if ctx == nil {
		t.Fatal("NewContext returned nil")
	}
	abs, _ := filepath.Abs(dir)
	if ctx.WorkDir != abs {
		t.Errorf("WorkDir = %s", ctx.WorkDir)
	}
	if ctx.ProjectID != "p1" {
		t.Errorf("ProjectID = %s", ctx.ProjectID)
	}
	if ctx.Trace == nil || ctx.Evidence == nil {
		t.Error("Trace and Evidence should be initialized")
	}
}

func TestNewContext_withBus(t *testing.T) {
	dir := t.TempDir()
	var b bus.Bus
	ctx, err := NewContext(Config{WorkDir: dir, Bus: b})
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	if ctx == nil {
		t.Fatal("NewContext returned nil")
	}
}

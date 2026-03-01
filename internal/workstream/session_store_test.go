package workstream

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewSessionStore(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewSessionStore(SessionStoreConfig{
		BasePath: tmpDir,
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	
	if store.ttl != time.Hour {
		t.Errorf("ttl = %v, want %v", store.ttl, time.Hour)
	}
}

func TestCreateWisp(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewSessionStore(SessionStoreConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	
	w := Wisp{
		Title:       "Test Wisp",
		Description: "Test description",
		Type:        "task",
		Priority:    2,
	}
	
	created, err := store.CreateWisp(w)
	if err != nil {
		t.Fatalf("CreateWisp failed: %v", err)
	}
	
	if created.ID == "" {
		t.Error("expected ID to be generated")
	}
	
	if created.Status != "open" {
		t.Errorf("Status = %q, want %q", created.Status, "open")
	}
	
	if created.ExpiresAt.IsZero() {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestGetWisp(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewSessionStore(SessionStoreConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	
	created, err := store.CreateWisp(Wisp{Title: "Test"})
	if err != nil {
		t.Fatalf("CreateWisp failed: %v", err)
	}
	
	got, err := store.GetWisp(created.ID)
	if err != nil {
		t.Fatalf("GetWisp failed: %v", err)
	}
	
	if got.Title != "Test" {
		t.Errorf("Title = %q, want %q", got.Title, "Test")
	}
}

func TestUpdateWispStatus(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewSessionStore(SessionStoreConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	
	created, err := store.CreateWisp(Wisp{Title: "Test"})
	if err != nil {
		t.Fatalf("CreateWisp failed: %v", err)
	}
	
	err = store.UpdateWispStatus(created.ID, "done")
	if err != nil {
		t.Fatalf("UpdateWispStatus failed: %v", err)
	}
	
	got, err := store.GetWisp(created.ID)
	if err != nil {
		t.Fatalf("GetWisp failed: %v", err)
	}
	
	if got.Status != "done" {
		t.Errorf("Status = %q, want %q", got.Status, "done")
	}
}

func TestListWisps(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewSessionStore(SessionStoreConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	
	// Create multiple wisps
	for i := 0; i < 3; i++ {
		_, err := store.CreateWisp(Wisp{Title: "Test"})
		if err != nil {
			t.Fatalf("CreateWisp %d failed: %v", i, err)
		}
	}
	
	wisps, err := store.ListWisps()
	if err != nil {
		t.Fatalf("ListWisps failed: %v", err)
	}
	
	if len(wisps) != 3 {
		t.Errorf("len(wisps) = %d, want 3", len(wisps))
	}
}

func TestListWispsExcludesExpired(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewSessionStore(SessionStoreConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	
	// Create an already-expired wisp
	w := Wisp{
		ID:        "expired-test",
		Title:     "Expired",
		ExpiresAt: time.Now().Add(-time.Hour),
		Status:    "open",
	}
	
	// Write directly to bypass auto-expiry
	wispPath := filepath.Join(tmpDir, "wisps", "expired-test.json")
	data, _ := json.MarshalIndent(w, "", "  ")
	os.MkdirAll(filepath.Dir(wispPath), 0755)
	os.WriteFile(wispPath, data, 0644)
	
	// Create a valid wisp
	_, err = store.CreateWisp(Wisp{Title: "Valid"})
	if err != nil {
		t.Fatalf("CreateWisp failed: %v", err)
	}
	
	wisps, err := store.ListWisps()
	if err != nil {
		t.Fatalf("ListWisps failed: %v", err)
	}
	
	if len(wisps) != 1 {
		t.Errorf("len(wisps) = %d, want 1 (expired should be excluded)", len(wisps))
	}
}

func TestClearSession(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewSessionStore(SessionStoreConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	
	// Create a wisp
	_, err = store.CreateWisp(Wisp{Title: "Test"})
	if err != nil {
		t.Fatalf("CreateWisp failed: %v", err)
	}
	
	// Clear
	if err := store.ClearSession(); err != nil {
		t.Fatalf("ClearSession failed: %v", err)
	}
	
	// Should be empty
	wisps, err := store.ListWisps()
	if err != nil {
		t.Fatalf("ListWisps failed: %v", err)
	}
	
	if len(wisps) != 0 {
		t.Errorf("len(wisps) = %d, want 0", len(wisps))
	}
}

func TestSessionStats(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewSessionStore(SessionStoreConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	
	// Create wisps with different types
	store.CreateWisp(Wisp{Title: "Task 1", Type: "task", Status: "open"})
	store.CreateWisp(Wisp{Title: "Task 2", Type: "task", Status: "done"})
	store.CreateWisp(Wisp{Title: "Bug 1", Type: "bug", Status: "open"})
	
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	
	if stats.ActiveWisps != 3 {
		t.Errorf("ActiveWisps = %d, want 3", stats.ActiveWisps)
	}
	
	if stats.ByStatus["open"] != 2 {
		t.Errorf("ByStatus[open] = %d, want 2", stats.ByStatus["open"])
	}
	
	if stats.ByType["task"] != 2 {
		t.Errorf("ByType[task] = %d, want 2", stats.ByType["task"])
	}
}

func TestGenerateWispID(t *testing.T) {
	store := &SessionStore{}
	
	id := store.generateWispID("test title", time.Now())
	
	if !strings.HasPrefix(id, "wisp-") {
		t.Errorf("ID = %q, should have wisp- prefix", id)
	}
	
	// Different titles should produce different IDs
	id2 := store.generateWispID("other title", time.Now())
	if id == id2 {
		t.Error("different titles should produce different IDs")
	}
}

package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestMain(m *testing.M) {
	os.Unsetenv("UPDATE_SNAPSHOTS")
	os.Exit(m.Run())
}

func TestCompareMatch(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, ".snapshots"))
	if err := s.Update("greeting", "hello world"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := s.Compare("greeting", "hello world"); err != nil {
		t.Fatalf("Compare should match: %v", err)
	}
}

func TestCompareMismatch(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, ".snapshots"))
	if err := s.Update("mismatch", "alpha\nbeta\n"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	err := s.Compare("mismatch", "alpha\ngamma\n")
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "- beta") || !strings.Contains(msg, "+ gamma") {
		t.Fatalf("expected diff lines in error, got: %v", msg)
	}
}

func TestCompareMissingSnapshot(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, ".snapshots"))
	err := s.Compare("nonexistent", "output")
	if err == nil {
		t.Fatal("expected error for missing snapshot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' in error, got: %v", err)
	}
}

func TestCompareCreatesWhenUpdateEnvSet(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, ".snapshots"))
	t.Setenv("UPDATE_SNAPSHOTS", "1")
	if err := s.Compare("auto", "content"); err != nil {
		t.Fatalf("Compare with UPDATE_SNAPSHOTS=1 should succeed: %v", err)
	}
	// Verify file was created.
	raw, err := os.ReadFile(s.snapshotPath("auto"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(raw) != "content" {
		t.Fatalf("expected %q, got %q", "content", string(raw))
	}
}

func TestUpdateCreatesFile(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, ".snapshots"))
	if err := s.Update("new", "data"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	raw, err := os.ReadFile(s.snapshotPath("new"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(raw) != "data" {
		t.Fatalf("expected %q, got %q", "data", string(raw))
	}
}

func TestUpdateOverwrites(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, ".snapshots"))
	if err := s.Update("rewrite", "v1"); err != nil {
		t.Fatalf("Update v1: %v", err)
	}
	if err := s.Update("rewrite", "v2"); err != nil {
		t.Fatalf("Update v2: %v", err)
	}
	raw, _ := os.ReadFile(s.snapshotPath("rewrite"))
	if string(raw) != "v2" {
		t.Fatalf("expected v2, got %q", string(raw))
	}
}

func TestUpdateCreatesNestedDirs(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, "a", "b", ".snapshots"))
	if err := s.Update("deep", "content"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	raw, _ := os.ReadFile(s.snapshotPath("deep"))
	if string(raw) != "content" {
		t.Fatalf("expected %q, got %q", "content", string(raw))
	}
}

func TestShouldUpdateEnvOn(t *testing.T) {
	s := New("")
	t.Setenv("UPDATE_SNAPSHOTS", "1")
	if !s.ShouldUpdate() {
		t.Fatal("expected ShouldUpdate=true when UPDATE_SNAPSHOTS=1")
	}
}

func TestShouldUpdateEnvOff(t *testing.T) {
	s := New("")
	t.Setenv("UPDATE_SNAPSHOTS", "0")
	if s.ShouldUpdate() {
		t.Fatal("expected ShouldUpdate=false when UPDATE_SNAPSHOTS=0")
	}
}

func TestShouldUpdateEnvUnset(t *testing.T) {
	s := New("")
	os.Unsetenv("UPDATE_SNAPSHOTS")
	if s.ShouldUpdate() {
		t.Fatal("expected ShouldUpdate=false when env unset")
	}
}

func TestDefaultSnapDir(t *testing.T) {
	s := New("")
	if s.SnapDir != ".snapshots" {
		t.Fatalf("expected .snapshots, got %q", s.SnapDir)
	}
}

func TestSnapshotPath(t *testing.T) {
	s := New("/tmp/snaps")
	p := s.snapshotPath("my-test")
	expected := filepath.Join("/tmp/snaps", "my-test.snap")
	if p != expected {
		t.Fatalf("expected %q, got %q", expected, p)
	}
}

func TestDiffLines(t *testing.T) {
	d := diffLines("a\nb\nc\n", "a\nx\nc\n")
	if !strings.Contains(d, "- b") || !strings.Contains(d, "+ x") {
		t.Fatalf("unexpected diff: %s", d)
	}
}

func TestUpdate_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, ".snapshots"))

	// Spawn multiple goroutines writing to the same snapshot name.
	const writers = 20
	var wg sync.WaitGroup
	wg.Add(writers)
	errCh := make(chan error, writers)

	for i := 0; i < writers; i++ {
		go func(i int) {
			defer wg.Done()
			content := strings.Repeat("x", 1000) + string(rune('A'+i%26))
			if err := s.Update("concurrent", content); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent Update failed: %v", err)
	}

	// Verify the file exists and is non-empty (one of the writes won).
	raw, err := os.ReadFile(s.snapshotPath("concurrent"))
	if err != nil {
		t.Fatalf("ReadFile after concurrent writes: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("snapshot file is empty after concurrent writes")
	}
}

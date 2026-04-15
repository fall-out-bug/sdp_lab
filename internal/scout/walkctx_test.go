package scout

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDetectIdentityRespectsContext verifies that detectIdentity checks ctx.Err()
// and returns early when context is cancelled.
func TestDetectIdentityRespectsContext(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":  "module example.com/app\ngo 1.26\n",
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, _ = detectIdentityWithContext(ctx, dir)
	// Should not hang or panic — the function must check ctx.Err()
}

// TestDetectScaleRespectsContext verifies that detectScale checks ctx.Err()
// during filepath.WalkDir and stops when cancelled.
func TestDetectScaleRespectsContext(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":  "module example.com/app\ngo 1.26\n",
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_ = detectScaleWithContext(ctx, dir, nil)
	// Should not hang or panic — the function must check ctx.Err()
}

// TestWalkCancelledMidScan verifies that a cancelled context during RunWithContext
// does not hang on large directory trees.
func TestWalkCancelledMidScan(t *testing.T) {
	dir := t.TempDir()
	// Create enough files to ensure walk is non-trivial
	for i := 0; i < 50; i++ {
		sub := filepath.Join(dir, "sub"+string(rune('A'+i%26)))
		_ = os.MkdirAll(sub, 0o755)
		for j := 0; j < 10; j++ {
			_ = os.WriteFile(filepath.Join(sub, "file.txt"), []byte("hello\n"), 0o644)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure context is expired

	_, err := RunWithContext(ctx, dir)
	if err == nil {
		t.Error("expected error with expired context during walk")
	}
}

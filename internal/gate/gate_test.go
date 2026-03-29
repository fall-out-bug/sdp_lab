package gate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGate_Status_Pending(t *testing.T) {
	g := &Gate{
		ID:        "abc12345",
		Question:  "Approve deployment?",
		CreatedAt: time.Now(),
	}
	if got := g.Status(); got != "pending" {
		t.Errorf("Status() = %q, want %q", got, "pending")
	}
}

func TestGate_Status_Resolved(t *testing.T) {
	now := time.Now()
	g := &Gate{
		ID:         "abc12345",
		Question:   "Approve deployment?",
		CreatedAt:  now.Add(-time.Minute),
		Answer:     "approve",
		Answerer:   "alice",
		ResolvedAt: &now,
	}
	if got := g.Status(); got != "resolved" {
		t.Errorf("Status() = %q, want %q", got, "resolved")
	}
}

func TestGate_Status_TimedOut(t *testing.T) {
	g := &Gate{
		ID:        "abc12345",
		Question:  "Approve deployment?",
		CreatedAt: time.Now().Add(-10 * time.Minute),
		Timeout:   5 * time.Minute,
	}
	if got := g.Status(); got != "timed_out" {
		t.Errorf("Status() = %q, want %q", got, "timed_out")
	}
}

func TestGate_IsBlocking(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		gate *Gate
		want bool
	}{
		{
			name: "pending is blocking",
			gate: &Gate{
				ID:        "aaa",
				Question:  "q?",
				CreatedAt: now,
			},
			want: true,
		},
		{
			name: "resolved is not blocking",
			gate: &Gate{
				ID:         "bbb",
				Question:   "q?",
				CreatedAt:  now.Add(-time.Minute),
				Answer:     "yes",
				Answerer:   "bob",
				ResolvedAt: &now,
			},
			want: false,
		},
		{
			name: "timed_out is not blocking",
			gate: &Gate{
				ID:        "ccc",
				Question:  "q?",
				CreatedAt: now.Add(-10 * time.Minute),
				Timeout:   5 * time.Minute,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gate.IsBlocking(); got != tt.want {
				t.Errorf("IsBlocking() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBeadsGateManager_CreateAndResolve(t *testing.T) {
	tmp := t.TempDir()
	mgr := &beadsGateManager{ProjectRoot: tmp}

	g, err := mgr.CreateGate("Deploy to prod?", "version 2.3", []string{"approve", "reject"})
	if err != nil {
		t.Fatalf("CreateGate: %v", err)
	}
	if g.ID == "" {
		t.Fatal("expected non-empty gate ID")
	}
	if g.Question != "Deploy to prod?" {
		t.Errorf("Question = %q, want %q", g.Question, "Deploy to prod?")
	}
	if g.Status() != "pending" {
		t.Errorf("new gate Status() = %q, want pending", g.Status())
	}

	// Verify file exists on disk
	path := filepath.Join(tmp, ".sdp", "gates", g.ID+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("gate file not found at %s", path)
	}

	// Resolve
	if err := mgr.ResolveGate(g.ID, "approve", "alice"); err != nil {
		t.Fatalf("ResolveGate: %v", err)
	}

	resolved, err := mgr.CheckGate(g.ID)
	if err != nil {
		t.Fatalf("CheckGate: %v", err)
	}
	if resolved.Status() != "resolved" {
		t.Errorf("resolved gate Status() = %q, want resolved", resolved.Status())
	}
	if resolved.Answer != "approve" {
		t.Errorf("Answer = %q, want approve", resolved.Answer)
	}
	if resolved.Answerer != "alice" {
		t.Errorf("Answerer = %q, want alice", resolved.Answerer)
	}
}

func TestBeadsGateManager_ListPending(t *testing.T) {
	tmp := t.TempDir()
	mgr := &beadsGateManager{ProjectRoot: tmp}

	// Create 3 gates
	g1, err := mgr.CreateGate("Q1?", "", nil)
	if err != nil {
		t.Fatalf("CreateGate g1: %v", err)
	}
	_, err = mgr.CreateGate("Q2?", "", nil)
	if err != nil {
		t.Fatalf("CreateGate g2: %v", err)
	}
	_, err = mgr.CreateGate("Q3?", "", nil)
	if err != nil {
		t.Fatalf("CreateGate g3: %v", err)
	}

	// Resolve one
	if err := mgr.ResolveGate(g1.ID, "done", "bob"); err != nil {
		t.Fatalf("ResolveGate: %v", err)
	}

	pending, err := mgr.ListPending()
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("ListPending returned %d gates, want 2", len(pending))
	}
}

func TestWaitForGate_AlreadyResolved(t *testing.T) {
	tmp := t.TempDir()
	mgr := &beadsGateManager{ProjectRoot: tmp}

	g, err := mgr.CreateGate("Q?", "", nil)
	if err != nil {
		t.Fatalf("CreateGate: %v", err)
	}
	if err := mgr.ResolveGate(g.ID, "yes", "charlie"); err != nil {
		t.Fatalf("ResolveGate: %v", err)
	}

	start := time.Now()
	result, err := waitForGate(context.Background(), mgr, g.ID, 100*time.Millisecond, 2*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("waitForGate: %v", err)
	}
	if result.Answer != "yes" {
		t.Errorf("Answer = %q, want yes", result.Answer)
	}
	// Should return nearly instantly
	if elapsed > 500*time.Millisecond {
		t.Errorf("waitForGate took %v, expected near-instant for already resolved gate", elapsed)
	}
}

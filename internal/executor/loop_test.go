package executor

import (
	"context"
	"testing"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/control"
)

func createReadyCardForLoop(t *testing.T, store *control.Store) *control.FeatureCard {
	t.Helper()
	card, err := store.CreateCard("openclaw", "Loop feature", "test loop")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "loop intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func TestRunOrchestrateLoopMaxCyclesOne(t *testing.T) {
	store := setupStore(t)
	_ = createReadyCardForLoop(t, store)

	control.SetCreateBeadsIssueFn(control.MockCreateBeadsIssue("bd-123"))
	defer control.SetCreateBeadsIssueFn(control.MockCreateBeadsIssue("bd-default"))

	if err := RunOrchestrateLoop(context.Background(), store, nil, store.ProjectRoot, 0, 1); err != nil {
		t.Fatalf("RunOrchestrateLoop error: %v", err)
	}

	card, err := store.SelectDispatchableCard()
	if err != nil {
		t.Fatal(err)
	}
	if card == nil || card.Status != "executing" {
		t.Fatalf("expected dispatched card to be executing, got %+v", card)
	}
}

func TestRunOrchestrateLoopNoDispatchableCards(t *testing.T) {
	store := setupStore(t)

	if err := RunOrchestrateLoop(context.Background(), store, nil, store.ProjectRoot, 0, 1); err != nil {
		t.Fatalf("RunOrchestrateLoop error: %v", err)
	}

	result, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "no_action" {
		t.Fatalf("action = %s, want no_action", result.Action)
	}
}

func TestRunOrchestrateLoopContextCancellation(t *testing.T) {
	store := setupStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := RunOrchestrateLoop(ctx, store, nil, store.ProjectRoot, time.Second, 0)
	if err != context.Canceled {
		t.Fatalf("error = %v, want %v", err, context.Canceled)
	}
}

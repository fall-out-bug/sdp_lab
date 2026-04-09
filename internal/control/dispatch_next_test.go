package control

import (
	"testing"
)

func TestSelectDispatchableCardReadyCard(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	selected, err := store.SelectDispatchableCard()
	if err != nil {
		t.Fatalf("SelectDispatchableCard error: %v", err)
	}

	if selected == nil {
		t.Fatal("expected to select ready card, got nil")
		return
	}
	if selected.ID != card.ID {
		t.Fatalf("selected card ID = %s, want %s", selected.ID, card.ID)
	}
	if selected.Status != "ready" {
		t.Fatalf("selected card status = %s, want ready", selected.Status)
	}
}

func TestSelectDispatchableCardNoReadyCards(t *testing.T) {
	store := setupStore(t)
	_, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	selected, err := store.SelectDispatchableCard()
	if err != nil {
		t.Fatalf("SelectDispatchableCard error: %v", err)
	}

	if selected != nil {
		t.Fatalf("expected nil (no ready cards), got card %s", selected.ID)
	}
}

func TestSelectDispatchableCardPrefersReadyOverExecuting(t *testing.T) {
	store := setupStore(t)

	readyCard, err := store.CreateCard("openclaw", "Ready feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	readyCard, err = store.ClarifyCard("openclaw", readyCard.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.MarkReady("openclaw", readyCard.ID)
	if err != nil {
		t.Fatal(err)
	}

	executingCard, err := store.CreateCard("openclaw", "Executing feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	executingCard, err = store.ClarifyCard("openclaw", executingCard.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.MarkReady("openclaw", executingCard.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ExecuteCard("openclaw", executingCard.ID)
	if err != nil {
		t.Fatal(err)
	}

	selected, err := store.SelectDispatchableCard()
	if err != nil {
		t.Fatalf("SelectDispatchableCard error: %v", err)
	}

	if selected == nil {
		t.Fatal("expected to select a card, got nil")
		return
	}
	if selected.ID != readyCard.ID {
		t.Fatalf("selected card ID = %s, want ready card %s (not executing)", selected.ID, readyCard.ID)
	}
}

func TestDispatchNextSuccess(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.DispatchNext()
	if err != nil {
		t.Fatalf("DispatchNext error: %v", err)
	}

	if !result.Success {
		t.Fatalf("Success = false, want true. Message: %s", result.Message)
	}
	if result.ProjectID != card.ProjectID {
		t.Fatalf("ProjectID = %s, want %s", result.ProjectID, card.ProjectID)
	}
	if result.CardID != card.ID {
		t.Fatalf("CardID = %s, want %s", result.CardID, card.ID)
	}
	if result.CardTitle != card.Title {
		t.Fatalf("CardTitle = %s, want %s", result.CardTitle, card.Title)
	}
	if result.ExecutorRole == "" {
		t.Fatal("ExecutorRole should be set")
	}
	if result.PacketPath == "" {
		t.Fatal("PacketPath should be set")
	}
	if result.NoDispatchableReason != "" {
		t.Fatalf("NoDispatchableReason should be empty, got %s", result.NoDispatchableReason)
	}
}

func TestDispatchNextNoDispatchable(t *testing.T) {
	store := setupStore(t)
	_, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.DispatchNext()
	if err != nil {
		t.Fatalf("DispatchNext error: %v", err)
	}

	if result.Success {
		t.Fatal("Success should be false when no dispatchable cards")
	}
	if result.ProjectID != "" || result.CardID != "" {
		t.Fatal("ProjectID and CardID should be empty when no dispatchable cards")
	}
	if result.ExecutorRole != "" || result.PacketPath != "" {
		t.Fatal("ExecutorRole and PacketPath should be empty when no dispatchable cards")
	}
	if result.NoDispatchableReason == "" {
		t.Fatal("NoDispatchableReason should be set when no dispatchable cards")
	}
}

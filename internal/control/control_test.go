package control

import (
	"os"
	"path/filepath"
	"testing"
)

func setupStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "specs"), 0o755); err != nil { t.Fatal(err) }
	registry := []byte("projects:\n  - id: openclaw\n    repo_url: https://github.com/openclaw/openclaw\n    beads_prefix: openclaw\n  - id: beads\n    repo_url: https://github.com/fall-out-bug/beads\n    beads_prefix: beads\n")
	if err := os.WriteFile(filepath.Join(root, "docs", "specs", "project-registry.yaml"), registry, 0o644); err != nil { t.Fatal(err) }
	store, err := Open(root)
	if err != nil { t.Fatal(err) }
	return store
}

func TestCreateCardCreatesCardAndIntakeArtifact(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil { t.Fatal(err) }
	if card.Status != "inbox" { t.Fatalf("status = %s", card.Status) }
	if len(card.IntakeArtifact) != 1 { t.Fatalf("expected intake artifact") }
	if _, err := os.Stat(filepath.Join(store.intakeDir("openclaw"), card.ID+".md")); err != nil { t.Fatal(err) }
	if _, err := os.Stat(store.cardPath("openclaw", card.ID)); err != nil { t.Fatal(err) }
}

func TestClarifyNeedsInputReadyAndPark(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil { t.Fatal(err) }
	card, err = store.ClarifyCard("openclaw", card.ID, "unify reminder policy", "feature", "openclaw", "medium", "ask one product question", []string{"escalation levels"}, []string{"calendar redesign"})
	if err != nil { t.Fatal(err) }
	if card.Status != "clarifying" || card.NormalizedIntent == "" { t.Fatalf("clarify failed: %+v", card) }
	card, err = store.MarkNeedsInput("openclaw", card.ID, []string{"author"}, []string{"Which channels?"}, []string{"Choose threshold model"}, []string{"one decision needed"}, nil)
	if err != nil { t.Fatal(err) }
	if card.Status != "needs_input" || len(card.NeedsFeedbackFrom) != 1 { t.Fatalf("needs_input failed: %+v", card) }
	card, err = store.MarkReady("openclaw", card.ID)
	if err != nil { t.Fatal(err) }
	if card.Status != "ready" { t.Fatalf("ready failed: %+v", card) }
	card, err = store.ParkCard("openclaw", card.ID, "deferred by owner")
	if err != nil { t.Fatal(err) }
	if card.Status != "parked" { t.Fatalf("park failed: %+v", card) }
}

func TestMarkReadyRequiresFields(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil { t.Fatal(err) }
	if _, err := store.MarkReady("openclaw", card.ID); err == nil { t.Fatal("expected validation error") }
}

func TestBuildSnapshots(t *testing.T) {
	store := setupStore(t)
	card1, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil { t.Fatal(err) }
	card1.Status = "needs_input"
	card1.NeedsFeedbackFrom = []string{"author"}
	card1.FeedbackRequest = []string{"Which channel?"}
	card1.AuthorUpdate = []string{"One decision needed"}
	if err := store.SaveCard(card1); err != nil { t.Fatal(err) }
	card2, err := store.CreateCard("beads", "Fix queue sorting", "fix queue order")
	if err != nil { t.Fatal(err) }
	card2.Status = "ready"
	card2.RecommendedNext = "spawn execution"
	if err := store.SaveCard(card2); err != nil { t.Fatal(err) }
	ps, err := store.BuildProjectSnapshot("openclaw")
	if err != nil { t.Fatal(err) }
	if ps.Counts["needs_input"] != 1 { t.Fatalf("needs_input count = %d", ps.Counts["needs_input"]) }
	port, err := store.BuildPortfolioSnapshot()
	if err != nil { t.Fatal(err) }
	if len(port.Queues["waiting_on_human"]) != 1 { t.Fatalf("waiting_on_human = %d", len(port.Queues["waiting_on_human"])) }
	if len(port.Queues["ready_to_execute"]) != 1 { t.Fatalf("ready_to_execute = %d", len(port.Queues["ready_to_execute"])) }
}

package control

import (
	"os"
	"path/filepath"
	"testing"
)

func setupStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := []byte("projects:\n  - id: openclaw\n    repo_url: https://github.com/openclaw/openclaw\n    beads_prefix: openclaw\n  - id: beads\n    repo_url: https://github.com/fall-out-bug/beads\n    beads_prefix: beads\n")
	if err := os.WriteFile(filepath.Join(root, "docs", "specs", "project-registry.yaml"), registry, 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestCreateCardCreatesCardAndIntakeArtifact(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil {
		t.Fatal(err)
	}
	if card.Status != "inbox" {
		t.Fatalf("status = %s", card.Status)
	}
	if len(card.IntakeArtifact) != 1 {
		t.Fatalf("expected intake artifact")
	}
	if _, err := os.Stat(filepath.Join(store.intakeDir("openclaw"), card.ID+".md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.cardPath("openclaw", card.ID)); err != nil {
		t.Fatal(err)
	}
}

func TestClarifyNeedsInputReadyAndPark(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "unify reminder policy", "feature", "openclaw", "medium", "ask one product question", []string{"escalation levels"}, []string{"calendar redesign"})
	if err != nil {
		t.Fatal(err)
	}
	if card.Status != "clarifying" || card.NormalizedIntent == "" {
		t.Fatalf("clarify failed: %+v", card)
	}
	card, err = store.MarkNeedsInput("openclaw", card.ID, []string{"author"}, []string{"Which channels?"}, []string{"Choose threshold model"}, []string{"one decision needed"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if card.Status != "needs_input" || len(card.NeedsFeedbackFrom) != 1 {
		t.Fatalf("needs_input failed: %+v", card)
	}
	card, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if card.Status != "ready" {
		t.Fatalf("ready failed: %+v", card)
	}
	card, err = store.ParkCard("openclaw", card.ID, "deferred by owner")
	if err != nil {
		t.Fatal(err)
	}
	if card.Status != "parked" {
		t.Fatalf("park failed: %+v", card)
	}
}

func TestMarkReadyRequiresFields(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkReady("openclaw", card.ID); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBuildSnapshots(t *testing.T) {
	store := setupStore(t)
	card1, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil {
		t.Fatal(err)
	}
	card1.Status = "needs_input"
	card1.NeedsFeedbackFrom = []string{"author"}
	card1.FeedbackRequest = []string{"Which channel?"}
	card1.AuthorUpdate = []string{"One decision needed"}
	if err := store.SaveCard(card1); err != nil {
		t.Fatal(err)
	}
	card2, err := store.CreateCard("beads", "Fix queue sorting", "fix queue order")
	if err != nil {
		t.Fatal(err)
	}
	card2.Status = "ready"
	card2.RecommendedNext = "spawn execution"
	if err := store.SaveCard(card2); err != nil {
		t.Fatal(err)
	}
	ps, err := store.BuildProjectSnapshot("openclaw")
	if err != nil {
		t.Fatal(err)
	}
	if ps.Counts["needs_input"] != 1 {
		t.Fatalf("needs_input count = %d", ps.Counts["needs_input"])
	}
	port, err := store.BuildPortfolioSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(port.Queues["waiting_on_human"]) != 1 {
		t.Fatalf("waiting_on_human = %d", len(port.Queues["waiting_on_human"]))
	}
	if len(port.Queues["ready_to_execute"]) != 1 {
		t.Fatalf("ready_to_execute = %d", len(port.Queues["ready_to_execute"]))
	}
}

func TestExecuteCard(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"scope in"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	defer SetCreateBeadsIssueFn(createBeadsIssue)
	SetCreateBeadsIssueFn(MockCreateBeadsIssue("bd-test-123"))

	executed, err := store.ExecuteCard("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if executed.Status != "executing" {
		t.Fatalf("status = %s, want executing", executed.Status)
	}
	if len(executed.LinkedBeadsIDs) != 1 {
		t.Fatalf("linked_beads_ids count = %d, want 1", len(executed.LinkedBeadsIDs))
	}
	if executed.LinkedBeadsIDs[0] != "bd-test-123" {
		t.Fatalf("beads id = %s, want bd-test-123", executed.LinkedBeadsIDs[0])
	}

	_, err = os.Stat(filepath.Join(store.projectSnapshotsDir("openclaw"), "board.json"))
	if err != nil {
		t.Fatalf("project snapshot not updated: %v", err)
	}
}

func TestExecuteCardFailsIfNotReady(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	defer SetCreateBeadsIssueFn(createBeadsIssue)
	SetCreateBeadsIssueFn(MockCreateBeadsIssue("bd-test-123"))

	_, err = store.ExecuteCard("openclaw", card.ID)
	if err == nil {
		t.Fatal("expected error for non-ready card")
	}
}

func TestGenerateFeedbackPacket(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	card.DecisionRequired = []string{"Choose threshold model"}
	card.AuthorUpdate = []string{"One decision needed"}
	card.BlockingReasons = []string{"Waiting for user input"}
	card.RecommendedNext = "Answer the question"
	card.WaitingOn = []string{"human"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.GenerateFeedbackPacket("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	if packet.CardID != card.ID {
		t.Fatalf("packet.CardID = %s, want %s", packet.CardID, card.ID)
	}
	if packet.CardTitle != card.Title {
		t.Fatalf("packet.CardTitle = %s, want %s", packet.CardTitle, card.Title)
	}
	if packet.ProjectID != card.ProjectID {
		t.Fatalf("packet.ProjectID = %s, want %s", packet.ProjectID, card.ProjectID)
	}
	if packet.Status != card.Status {
		t.Fatalf("packet.Status = %s, want %s", packet.Status, card.Status)
	}
	if len(packet.NeedsFeedbackFrom) != 1 {
		t.Fatalf("len(NeedsFeedbackFrom) = %d, want 1", len(packet.NeedsFeedbackFrom))
	}
	if len(packet.FeedbackRequest) != 1 {
		t.Fatalf("len(FeedbackRequest) = %d, want 1", len(packet.FeedbackRequest))
	}
	if len(packet.DecisionRequired) != 1 {
		t.Fatalf("len(DecisionRequired) = %d, want 1", len(packet.DecisionRequired))
	}
	if len(packet.BlockingReasons) != 1 {
		t.Fatalf("len(BlockingReasons) = %d, want 1", len(packet.BlockingReasons))
	}
}

func TestGenerateFeedbackPacketFailsForInvalidStatus(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.GenerateFeedbackPacket("openclaw", card.ID); err == nil {
		t.Fatal("expected error for non-needs_input/blocked card")
	}
}

func TestApplyFeedback(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	card.BlockingReasons = []string{"Waiting for channel decision"}
	card.WaitingOn = []string{"human"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	answer := &FeedbackAnswer{
		FeedbackAnswers: []string{"Use chat only"},
		UnblockReasons:  []string{"Waiting for channel decision"},
	}

	resumed, err := store.ApplyFeedback("openclaw", card.ID, answer)
	if err != nil {
		t.Fatal(err)
	}

	if resumed.Status != "clarifying" {
		t.Fatalf("resumed.Status = %s, want clarifying", resumed.Status)
	}
	if len(resumed.NeedsFeedbackFrom) != 0 {
		t.Fatalf("len(NeedsFeedbackFrom) = %d, want 0", len(resumed.NeedsFeedbackFrom))
	}
	if len(resumed.FeedbackRequest) != 0 {
		t.Fatalf("len(FeedbackRequest) = %d, want 0", len(resumed.FeedbackRequest))
	}
	if len(resumed.BlockingReasons) != 0 {
		t.Fatalf("len(BlockingReasons) = %d, want 0", len(resumed.BlockingReasons))
	}
	if len(resumed.WaitingOn) != 0 {
		t.Fatalf("len(WaitingOn) = %d, want 0", len(resumed.WaitingOn))
	}
	if len(resumed.AuthorUpdate) == 0 {
		t.Fatal("expected author_update to contain answer")
	}
}

func TestApplyFeedbackWithReadyTarget(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.NormalizedIntent = "test intent"
	card.TaskType = "feature"
	card.TargetRepo = "openclaw"
	card.RiskLevel = "low"
	card.RecommendedNext = "execute"
	card.ScopeIn = []string{"test scope"}
	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Any questions?"}
	card.WaitingOn = []string{"human"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	answer := &FeedbackAnswer{
		FeedbackAnswers:    []string{"No questions"},
		ResumeTargetStatus: "ready",
	}

	resumed, err := store.ApplyFeedback("openclaw", card.ID, answer)
	if err != nil {
		t.Fatal(err)
	}

	if resumed.Status != "ready" {
		t.Fatalf("resumed.Status = %s, want ready", resumed.Status)
	}
}

func TestApplyFeedbackFailsForInvalidTarget(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.NormalizedIntent = "test intent"
	card.TaskType = "feature"
	card.TargetRepo = "openclaw"
	card.RiskLevel = "low"
	card.RecommendedNext = "execute"
	card.ScopeIn = []string{"test scope"}
	card.Status = "needs_input"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	answer := &FeedbackAnswer{
		FeedbackAnswers:    []string{"No questions"},
		ResumeTargetStatus: "ready",
	}

	card.NormalizedIntent = ""
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	if _, err := store.ApplyFeedback("openclaw", card.ID, answer); err == nil {
		t.Fatal("expected error for card not meeting ready gate")
	}
}

func TestApplyFeedbackFailsForInvalidStatus(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	answer := &FeedbackAnswer{
		FeedbackAnswers: []string{"test"},
	}

	if _, err := store.ApplyFeedback("openclaw", card.ID, answer); err == nil {
		t.Fatal("expected error for non-needs_input/blocked card")
	}
}

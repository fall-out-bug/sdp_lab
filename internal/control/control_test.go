package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
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

func overwriteCard(t *testing.T, store *Store, card *FeatureCard) {
	t.Helper()
	data, err := yaml.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.cardPath(card.ProjectID, card.ID), data, 0o644); err != nil {
		t.Fatal(err)
	}
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

func TestExportFeedbackPacket(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	exportPath := filepath.Join(t.TempDir(), "feedback-packet.json")
	packet, err := store.ExportFeedbackPacket("openclaw", card.ID, exportPath)
	if err != nil {
		t.Fatal(err)
	}

	if packet.CardID != card.ID {
		t.Fatalf("packet.CardID = %s, want %s", packet.CardID, card.ID)
	}

	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("exported file not found: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}

	var importedPacket FeedbackPacket
	if err := json.Unmarshal(data, &importedPacket); err != nil {
		t.Fatalf("failed to parse exported packet: %v", err)
	}

	if importedPacket.CardID != card.ID {
		t.Fatalf("imported packet.CardID = %s, want %s", importedPacket.CardID, card.ID)
	}
}

func TestImportFeedbackAnswer(t *testing.T) {
	answer := &FeedbackAnswer{
		FeedbackAnswers:    []string{"Use chat only"},
		DecisionAnswers:    []string{"Threshold: high"},
		AuthorUpdates:      []string{"Update notes"},
		AdminActions:       []string{"Approved"},
		UnblockReasons:     []string{"Decision made"},
		ResumeTargetStatus: "ready",
	}

	answerPath := filepath.Join(t.TempDir(), "feedback-answer.json")
	data, err := json.MarshalIndent(answer, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answerPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	imported, err := ImportFeedbackAnswer(answerPath)
	if err != nil {
		t.Fatalf("import feedback answer: %v", err)
	}

	if len(imported.FeedbackAnswers) != 1 || imported.FeedbackAnswers[0] != "Use chat only" {
		t.Fatalf("FeedbackAnswers = %v, want [Use chat only]", imported.FeedbackAnswers)
	}
	if len(imported.DecisionAnswers) != 1 || imported.DecisionAnswers[0] != "Threshold: high" {
		t.Fatalf("DecisionAnswers = %v, want [Threshold: high]", imported.DecisionAnswers)
	}
	if imported.ResumeTargetStatus != "ready" {
		t.Fatalf("ResumeTargetStatus = %s, want ready", imported.ResumeTargetStatus)
	}
}

func TestImportFeedbackAnswerFailsForInvalidFile(t *testing.T) {
	invalidPath := filepath.Join(t.TempDir(), "nonexistent.json")
	if _, err := ImportFeedbackAnswer(invalidPath); err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	badJSONPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badJSONPath, []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportFeedbackAnswer(badJSONPath); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGenerateCorrelationID(t *testing.T) {
	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	if id1 == "" {
		t.Fatal("correlation id should not be empty")
	}
	if id2 == "" {
		t.Fatal("correlation id should not be empty")
	}
	if id1 == id2 {
		t.Fatalf("correlation ids should be unique: %s == %s", id1, id2)
	}
	if len(id1) < 10 {
		t.Fatalf("correlation id too short: %s", id1)
	}
}

func TestExportOutboundMessage(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	envelope, err := store.ExportOutboundMessage("openclaw", card.ID, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if envelope.CardID != card.ID {
		t.Fatalf("envelope.CardID = %s, want %s", envelope.CardID, card.ID)
	}
	if envelope.ProjectID != card.ProjectID {
		t.Fatalf("envelope.ProjectID = %s, want %s", envelope.ProjectID, card.ProjectID)
	}
	if envelope.CorrelationID == "" {
		t.Fatal("correlation_id should not be empty")
	}
	if envelope.TargetRole != "admin" {
		t.Fatalf("envelope.TargetRole = %s, want admin", envelope.TargetRole)
	}
	if envelope.Payload == nil {
		t.Fatal("payload should not be nil")
	}
	if envelope.Payload.CardID != card.ID {
		t.Fatalf("payload.CardID = %s, want %s", envelope.Payload.CardID, card.ID)
	}
}

func TestExportOutboundMessageFailsForInvalidStatus(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.ExportOutboundMessage("openclaw", card.ID, "human"); err == nil {
		t.Fatal("expected error for non-needs_input/blocked card")
	}
}

func TestIngestReplyWithReplyText(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	card.BlockingReasons = []string{"Waiting for input"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	envelope := &InboundReplyEnvelope{
		CardID:             card.ID,
		ProjectID:          "openclaw",
		CorrelationID:      "corr-123",
		ReplyText:          "Use chat only",
		ResumeTargetStatus: "clarifying",
	}

	resumed, err := store.IngestReply(envelope)
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
	if len(resumed.AuthorUpdate) == 0 {
		t.Fatal("expected author_update to contain reply")
	}
}

func TestIngestReplyWithAnswers(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	card.BlockingReasons = []string{"Waiting for input"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	envelope := &InboundReplyEnvelope{
		CardID:             card.ID,
		ProjectID:          "openclaw",
		CorrelationID:      "corr-456",
		Answers:            []string{"Chat only", "High priority"},
		ResumeTargetStatus: "clarifying",
	}

	resumed, err := store.IngestReply(envelope)
	if err != nil {
		t.Fatal(err)
	}

	if resumed.Status != "clarifying" {
		t.Fatalf("resumed.Status = %s, want clarifying", resumed.Status)
	}
	if len(resumed.AuthorUpdate) == 0 {
		t.Fatal("expected author_update to contain answers")
	}
}

func TestIngestReplyWithReadyTarget(t *testing.T) {
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
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	envelope := &InboundReplyEnvelope{
		CardID:             card.ID,
		ProjectID:          "openclaw",
		CorrelationID:      "corr-789",
		ReplyText:          "No questions",
		ResumeTargetStatus: "ready",
	}

	resumed, err := store.IngestReply(envelope)
	if err != nil {
		t.Fatal(err)
	}

	if resumed.Status != "ready" {
		t.Fatalf("resumed.Status = %s, want ready", resumed.Status)
	}
}

func TestIngestReplyFailsForMissingCardID(t *testing.T) {
	store := setupStore(t)

	envelope := &InboundReplyEnvelope{
		ProjectID:     "openclaw",
		CorrelationID: "corr-123",
		ReplyText:     "test",
	}

	if _, err := store.IngestReply(envelope); err == nil {
		t.Fatal("expected error for missing card_id")
	}
}

func TestIngestReplyFailsForMissingProjectID(t *testing.T) {
	store := setupStore(t)

	envelope := &InboundReplyEnvelope{
		CardID:        "test-card",
		CorrelationID: "corr-123",
		ReplyText:     "test",
	}

	if _, err := store.IngestReply(envelope); err == nil {
		t.Fatal("expected error for missing project_id")
	}
}

func TestIngestReplyRoundtrip(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	outbound, err := store.ExportOutboundMessage("openclaw", card.ID, "human")
	if err != nil {
		t.Fatal(err)
	}

	inbound := &InboundReplyEnvelope{
		CardID:             outbound.CardID,
		ProjectID:          outbound.ProjectID,
		CorrelationID:      outbound.CorrelationID,
		ReplyText:          "Use email",
		ResumeTargetStatus: "clarifying",
	}

	resumed, err := store.IngestReply(inbound)
	if err != nil {
		t.Fatal(err)
	}

	if resumed.Status != "clarifying" {
		t.Fatalf("resumed.Status = %s, want clarifying", resumed.Status)
	}
	if len(resumed.AuthorUpdate) == 0 {
		t.Fatal("expected author_update to contain reply")
	}
}

func TestAttentionCommandQueues(t *testing.T) {
	store := setupStore(t)

	card1, err := store.CreateCard("openclaw", "Test feedback", "need input")
	if err != nil {
		t.Fatal(err)
	}
	card1.Status = "needs_input"
	card1.NeedsFeedbackFrom = []string{"author"}
	card1.FeedbackRequest = []string{"Which channel?"}
	card1.RecommendedNext = "Answer question"
	if err := store.SaveCard(card1); err != nil {
		t.Fatal(err)
	}

	card2, err := store.CreateCard("beads", "Test ready", "ready card")
	if err != nil {
		t.Fatal(err)
	}
	card2.Status = "ready"
	card2.RecommendedNext = "Execute"
	if err := store.SaveCard(card2); err != nil {
		t.Fatal(err)
	}

	card3, err := store.CreateCard("openclaw", "Test blocked", "blocked card")
	if err != nil {
		t.Fatal(err)
	}
	card3.Status = "blocked"
	card3.BlockingReasons = []string{"Waiting for decision"}
	card3.AdminActionRequired = []string{"Approve scope"}
	card3.RecommendedNext = "Resolve blocker"
	if err := store.SaveCard(card3); err != nil {
		t.Fatal(err)
	}

	card4, err := store.CreateCard("beads", "Test executing", "executing card")
	if err != nil {
		t.Fatal(err)
	}
	card4.Status = "executing"
	card4.ActiveAgents = []string{"executor"}
	card4.LinkedBeadsIDs = []string{"bd-123"}
	card4.RecommendedNext = "Complete work"
	if err := store.SaveCard(card4); err != nil {
		t.Fatal(err)
	}

	snap, err := store.BuildPortfolioSnapshot()
	if err != nil {
		t.Fatal(err)
	}

	if len(snap.Queues["waiting_on_human"]) != 1 {
		t.Fatalf("waiting_on_human count = %d, want 1", len(snap.Queues["waiting_on_human"]))
	}
	if len(snap.Queues["ready_to_execute"]) != 1 {
		t.Fatalf("ready_to_execute count = %d, want 1", len(snap.Queues["ready_to_execute"]))
	}
	if len(snap.Queues["blocked"]) != 1 {
		t.Fatalf("blocked count = %d, want 1", len(snap.Queues["blocked"]))
	}

	waiting := snap.Queues["waiting_on_human"][0]
	if waiting.ProjectID != "openclaw" {
		t.Fatalf("waiting.ProjectID = %s, want openclaw", waiting.ProjectID)
	}
	if waiting.CardID != card1.ID {
		t.Fatalf("waiting.CardID = %s, want %s", waiting.CardID, card1.ID)
	}
	if len(waiting.ActiveAgents) == 0 {
		t.Fatal("expected ActiveAgents in queue item")
	}

	ready := snap.Queues["ready_to_execute"][0]
	if ready.ProjectID != "beads" {
		t.Fatalf("ready.ProjectID = %s, want beads", ready.ProjectID)
	}

	blocked := snap.Queues["blocked"][0]
	if len(blocked.AdminActionRequired) == 0 {
		t.Fatal("expected AdminActionRequired in blocked item")
	}

	if snap.NextAction["recommended"] != "surface_feedback_request" {
		t.Fatalf("NextAction.recommended = %s, want surface_feedback_request", snap.NextAction["recommended"])
	}
}

func TestAttentionCommandNextAction(t *testing.T) {
	store := setupStore(t)

	card, err := store.CreateCard("openclaw", "Test next", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	snap, err := store.BuildPortfolioSnapshot()
	if err != nil {
		t.Fatal(err)
	}

	if snap.NextAction["recommended"] != "surface_feedback_request" {
		t.Fatalf("NextAction.recommended = %s, want surface_feedback_request", snap.NextAction["recommended"])
	}
	if snap.NextAction["target_project_id"] != "openclaw" {
		t.Fatalf("NextAction.target_project_id = %s, want openclaw", snap.NextAction["target_project_id"])
	}
}

func TestAttentionCommandExecutingCards(t *testing.T) {
	store := setupStore(t)

	card, err := store.CreateCard("openclaw", "Test executing", "executing")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "executing"
	card.ActiveAgents = []string{"executor"}
	card.LinkedBeadsIDs = []string{"bd-456"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	projSnap, err := store.BuildProjectSnapshot("openclaw")
	if err != nil {
		t.Fatal(err)
	}

	if len(projSnap.Columns["executing"]) != 1 {
		t.Fatalf("executing count = %d, want 1", len(projSnap.Columns["executing"]))
	}

	executingCard := projSnap.Columns["executing"][0]
	if len(executingCard.ActiveAgents) == 0 {
		t.Fatal("expected ActiveAgents in executing card")
	}
	if len(executingCard.LinkedBeadsIDs) == 0 {
		t.Fatal("expected LinkedBeadsIDs in executing card")
	}
}

func TestDoctorControlDetectsMissingIntakeArtifact(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.IntakeArtifact = nil
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 1 {
		t.Fatalf("TotalChecks = %d, want 1", report.TotalChecks)
	}
	if report.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", report.Failed)
	}
	if len(report.Checks) != 1 {
		t.Fatalf("len(Checks) = %d, want 1", len(report.Checks))
	}
	if report.Checks[0].CheckID != "missing-intake-artifact" {
		t.Fatalf("CheckID = %s, want missing-intake-artifact", report.Checks[0].CheckID)
	}
}

func TestDoctorControlDetectsIntakeArtifactNotFound(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.IntakeArtifact = []string{"/nonexistent/path.md"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 1 {
		t.Fatalf("TotalChecks = %d, want 1", report.TotalChecks)
	}
	if report.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", report.Failed)
	}
	if len(report.Checks) != 1 {
		t.Fatalf("len(Checks) = %d, want 1", len(report.Checks))
	}
	if report.Checks[0].CheckID != "intake-artifact-not-found" {
		t.Fatalf("CheckID = %s, want intake-artifact-not-found", report.Checks[0].CheckID)
	}
}

func TestDoctorControlDetectsReadyGateMissing(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 1 {
		t.Fatalf("TotalChecks = %d, want 1", report.TotalChecks)
	}
	if report.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", report.Failed)
	}
	if len(report.Checks) != 1 {
		t.Fatalf("len(Checks) = %d, want 1", len(report.Checks))
	}
	if report.Checks[0].CheckID != "ready-gate-missing" {
		t.Fatalf("CheckID = %s, want ready-gate-missing", report.Checks[0].CheckID)
	}
}

func TestDoctorControlDetectsExecutingWithoutBeads(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "executing"
	card.LinkedBeadsIDs = nil
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 1 {
		t.Fatalf("TotalChecks = %d, want 1", report.TotalChecks)
	}
	if report.Failed < 1 {
		t.Fatalf("Failed = %d, want at least 1", report.Failed)
	}
	found := false
	for _, check := range report.Checks {
		if check.CheckID == "executing-without-beads" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected executing-without-beads in checks: %+v", report.Checks)
	}
}

func TestDoctorControlDetectsNeedsInputWithoutQuestions(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 1 {
		t.Fatalf("TotalChecks = %d, want 1", report.TotalChecks)
	}
	if report.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", report.Failed)
	}
	if len(report.Checks) != 1 {
		t.Fatalf("len(Checks) = %d, want 1", len(report.Checks))
	}
	if report.Checks[0].CheckID != "needs-input-without-questions" {
		t.Fatalf("CheckID = %s, want needs-input-without-questions", report.Checks[0].CheckID)
	}
}

func TestDoctorControlDetectsStaleReadyCard(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"scope in"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}
	card.UpdatedAt = time.Now().Add(-73 * time.Hour).UTC().Format(time.RFC3339)
	overwriteCard(t, store, card)

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.Failed != 1 || len(report.Checks) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Checks[0].CheckID != "stale-ready-card" {
		t.Fatalf("CheckID = %s, want stale-ready-card", report.Checks[0].CheckID)
	}
	if report.Checks[0].Severity != "warning" {
		t.Fatalf("Severity = %s, want warning", report.Checks[0].Severity)
	}
}

func TestDoctorControlDetectsStaleNeedsInputCard(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	card.UpdatedAt = time.Now().Add(-49 * time.Hour).UTC().Format(time.RFC3339)
	overwriteCard(t, store, card)

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.Failed != 1 || len(report.Checks) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Checks[0].CheckID != "stale-needs-input-card" {
		t.Fatalf("CheckID = %s, want stale-needs-input-card", report.Checks[0].CheckID)
	}
}

func TestDoctorControlDetectsStaleBlockedCard(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "blocked"
	card.BlockingReasons = []string{"waiting for decision"}
	card.UpdatedAt = time.Now().Add(-73 * time.Hour).UTC().Format(time.RFC3339)
	overwriteCard(t, store, card)

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.Failed != 1 || len(report.Checks) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Checks[0].CheckID != "stale-blocked-card" {
		t.Fatalf("CheckID = %s, want stale-blocked-card", report.Checks[0].CheckID)
	}
}

func TestDoctorControlDetectsExecutingWithoutDispatchMetadata(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "executing"
	card.LinkedBeadsIDs = []string{"bd-test-123"}
	overwriteCard(t, store, card)

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.Failed != 1 || len(report.Checks) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Checks[0].CheckID != "executing-without-dispatch-metadata" {
		t.Fatalf("CheckID = %s, want executing-without-dispatch-metadata", report.Checks[0].CheckID)
	}
	if report.Checks[0].Severity != "warning" {
		t.Fatalf("Severity = %s, want warning", report.Checks[0].Severity)
	}
}

func TestDoctorControlDetectsDoneWithoutResultSummary(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "done"
	overwriteCard(t, store, card)

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.Failed != 1 || len(report.Checks) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Checks[0].CheckID != "done-without-result-summary" {
		t.Fatalf("CheckID = %s, want done-without-result-summary", report.Checks[0].CheckID)
	}
}

func TestDoctorControlPassesHealthyCards(t *testing.T) {
	store := setupStore(t)

	_, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.CreateCard("beads", "Another feature", "another test")
	if err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 2 {
		t.Fatalf("TotalChecks = %d, want 2", report.TotalChecks)
	}
	if report.Passed != 2 {
		t.Fatalf("Passed = %d, want 2", report.Passed)
	}
	if report.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", report.Failed)
	}
	if len(report.Checks) != 0 {
		t.Fatalf("len(Checks) = %d, want 0", len(report.Checks))
	}
}

func TestDoctorControlMixedResults(t *testing.T) {
	store := setupStore(t)

	_, err := store.CreateCard("openclaw", "Healthy feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	badIntakeCard, err := store.CreateCard("openclaw", "Missing intake", "test")
	if err != nil {
		t.Fatal(err)
	}
	badIntakeCard.IntakeArtifact = nil
	if err := store.SaveCard(badIntakeCard); err != nil {
		t.Fatal(err)
	}

	readyCard, err := store.CreateCard("openclaw", "Bad ready", "test")
	if err != nil {
		t.Fatal(err)
	}
	readyCard.Status = "ready"
	if err := store.SaveCard(readyCard); err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 3 {
		t.Fatalf("TotalChecks = %d, want 3", report.TotalChecks)
	}
	if report.Passed != 1 {
		t.Fatalf("Passed = %d, want 1", report.Passed)
	}
	if report.Failed != 2 {
		t.Fatalf("Failed = %d, want 2", report.Failed)
	}
	if len(report.Checks) != 2 {
		t.Fatalf("len(Checks) = %d, want 2", len(report.Checks))
	}
}

func TestDoctorControlExecutingCardWithBeadsPasses(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "executing"
	card.LinkedBeadsIDs = []string{"bd-test-123"}
	card.DispatchedAt = time.Now().UTC().Format(time.RFC3339)
	card.DispatchedTo = "beads"
	card.DispatchedPacketPath = "/tmp/dispatch.json"
	overwriteCard(t, store, card)

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 1 {
		t.Fatalf("TotalChecks = %d, want 1", report.TotalChecks)
	}
	if report.Passed != 1 {
		t.Fatalf("Passed = %d, want 1", report.Passed)
	}
	if report.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", report.Failed)
	}
}

func TestDoctorControlNeedsInputWithQuestionsPasses(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	card.Status = "needs_input"
	card.NeedsFeedbackFrom = []string{"author"}
	card.FeedbackRequest = []string{"Which channel?"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 1 {
		t.Fatalf("TotalChecks = %d, want 1", report.TotalChecks)
	}
	if report.Passed != 1 {
		t.Fatalf("Passed = %d, want 1", report.Passed)
	}
	if report.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", report.Failed)
	}
}

func TestDoctorControlReadyCardWithAllFieldsPasses(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"scope in"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalChecks != 1 {
		t.Fatalf("TotalChecks = %d, want 1", report.TotalChecks)
	}
	if report.Passed != 1 {
		t.Fatalf("Passed = %d, want 1", report.Passed)
	}
	if report.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", report.Failed)
	}
}

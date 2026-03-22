package control

import (
	"testing"
)

func TestRouteToExecutorRoutesToClarificationForNeedsInput(t *testing.T) {
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

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleClarification {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleClarification)
	}
}

func TestRouteToExecutorRoutesToClarificationForClarifying(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "clarifying"
	card.OpenQuestions = []string{"What's the scope?"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleClarification {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleClarification)
	}
}

func TestRouteToExecutorRoutesToReviewForReviewing(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "reviewing"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleReview {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleReview)
	}
}

func TestRouteToExecutorRoutesToHumanAdminForHighRiskWithDecisions(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.RiskLevel = "high"
	card.DecisionRequired = []string{"Approve scope expansion"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleHumanAdmin {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleHumanAdmin)
	}
}

func TestRouteToExecutorRoutesToHumanAdminForHighRiskWithAdminActions(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.RiskLevel = "high"
	card.AdminActionRequired = []string{"Security review required"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleHumanAdmin {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleHumanAdmin)
	}
}

func TestRouteToExecutorRoutesToReleaseCheckForHighRiskRelease(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.RiskLevel = "high"
	card.ScopeIn = []string{"release", "migration"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleReleaseCheck {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleReleaseCheck)
	}
}

func TestRouteToExecutorRoutesToDocsTranslationForDocsScope(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.ScopeIn = []string{"docs", "translation"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleDocsTranslation {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleDocsTranslation)
	}
}

func TestRouteToExecutorRoutesToRepoLocalArchitectureForArchitecture(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.TargetArea = "architecture"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleRepoLocalArchitecture {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleRepoLocalArchitecture)
	}
}

func TestRouteToExecutorRoutesToOmOImplementationDefault(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "ready"
	card.RiskLevel = "low"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	role, err := store.RouteToExecutor(card)
	if err != nil {
		t.Fatalf("RouteToExecutor error: %v", err)
	}
	if role != ExecutorRoleOmOImplementation {
		t.Fatalf("executor role = %s, want %s", role, ExecutorRoleOmOImplementation)
	}
}

func TestRouteToExecutorFailsForNilCard(t *testing.T) {
	store := setupStore(t)
	_, err := store.RouteToExecutor(nil)
	if err == nil {
		t.Fatal("expected error for nil card")
	}
}

func TestBuildExecutionPacketForReadyCard(t *testing.T) {
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
	card.ScopeIn = []string{"implementation"}
	card.ScopeOut = []string{"code changes"}
	card.RequiredArtifacts = []string{"test coverage", "docs"}
	card.RequiredChecks = []string{"quality gates"}
	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	if packet.ParentFeatureID != card.ID {
		t.Fatalf("ParentFeatureID = %s, want %s", packet.ParentFeatureID, card.ID)
	}
	if packet.ProjectID != card.ProjectID {
		t.Fatalf("ProjectID = %s, want %s", packet.ProjectID, card.ProjectID)
	}
	if packet.TargetRepo != card.TargetRepo {
		t.Fatalf("TargetRepo = %s, want %s", packet.TargetRepo, card.TargetRepo)
	}
	if packet.ExecutorRole != string(ExecutorRoleOmOImplementation) {
		t.Fatalf("ExecutorRole = %s, want %s", packet.ExecutorRole, ExecutorRoleOmOImplementation)
	}
	if packet.Objective != card.NormalizedIntent {
		t.Fatalf("Objective = %s, want %s", packet.Objective, card.NormalizedIntent)
	}
	if len(packet.ScopeIn) == 0 {
		t.Fatal("ScopeIn should not be empty")
	}
	if len(packet.ScopeOut) == 0 {
		t.Fatal("ScopeOut should not be empty")
	}
}

func TestBuildExecutionPacketForCardWithLinkedBeads(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.LinkedBeadsIDs = []string{"bd-test-456"}
	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	if packet.BeadsTaskID != "bd-test-456" {
		t.Fatalf("BeadsTaskID = %s, want bd-test-456", packet.BeadsTaskID)
	}
}

func TestBuildExecutionPacketUsesRawRequestIfNoNormalizedIntent(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "raw request text")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	if packet.Objective != "raw request text" {
		t.Fatalf("Objective = %s, want 'raw request text'", packet.Objective)
	}
}

func TestBuildExecutionPacketIncludesRiskInConstraints(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.RiskLevel = "high"
	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	hasRiskConstraint := false
	for _, c := range packet.Constraints {
		if c == "risk_level: high" {
			hasRiskConstraint = true
			break
		}
	}
	if !hasRiskConstraint {
		t.Fatal("Constraints should include risk_level: high")
	}
}

func TestBuildExecutionPacketIncludesExecutionModeInConstraints(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.ExecutionMode = "tdd"
	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	hasExecutionModeConstraint := false
	for _, c := range packet.Constraints {
		if c == "execution_mode: tdd" {
			hasExecutionModeConstraint = true
			break
		}
	}
	if !hasExecutionModeConstraint {
		t.Fatal("Constraints should include execution_mode: tdd")
	}
}

func TestBuildExecutionPacketIncludesNonGoalsInConstraints(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.NonGoals = []string{"no UI changes", "no database migration"}
	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	hasNonGoalsConstraint := false
	for _, c := range packet.Constraints {
		if c == "non_goals: exclude from implementation" {
			hasNonGoalsConstraint = true
			break
		}
	}
	if !hasNonGoalsConstraint {
		t.Fatal("Constraints should include non_goals exclusion")
	}
}

func TestBuildExecutionPacketSetsNextHandoffFromRecommendedNext(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.RecommendedNext = "reviewer"
	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	if packet.NextHandoffTarget != "reviewer" {
		t.Fatalf("NextHandoffTarget = %s, want reviewer", packet.NextHandoffTarget)
	}
}

func TestBuildExecutionPacketSetsNextHandoffForReadyStatus(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "ready"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	if packet.NextHandoffTarget != "executor" {
		t.Fatalf("NextHandoffTarget = %s, want executor", packet.NextHandoffTarget)
	}
}

func TestBuildExecutionPacketSetsNextHandoffToOrchestratorDefault(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "clarifying"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet, err := store.BuildExecutionPacket("openclaw", card.ID)
	if err != nil {
		t.Fatalf("BuildExecutionPacket error: %v", err)
	}

	if packet.NextHandoffTarget != "orchestrator" {
		t.Fatalf("NextHandoffTarget = %s, want orchestrator", packet.NextHandoffTarget)
	}
}

func TestBuildExecutionPacketFailsForNonexistentCard(t *testing.T) {
	store := setupStore(t)
	_, err := store.BuildExecutionPacket("openclaw", "nonexistent-card-id")
	if err == nil {
		t.Fatal("expected error for nonexistent card")
	}
}

package evaluator

import (
	"errors"
	"testing"
)

func TestBuildPersonaExecutionPacketDeterministic(t *testing.T) {
	plan := DefaultDeepThinkingSwarmPlan()

	packet, err := BuildPersonaExecutionPacket("sdp_dev-hx0.1.2", plan)
	if err != nil {
		t.Fatalf("unexpected error building packet: %v", err)
	}

	if packet.ContractVersion != PersonaExecutionPacketContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", packet.ContractVersion, PersonaExecutionPacketContractVersion)
	}
	if packet.IssueID != "sdp_dev-hx0.1.2" {
		t.Fatalf("unexpected issue id: %s", packet.IssueID)
	}
	if packet.Cadence != "weekly-or-change-triggered" {
		t.Fatalf("unexpected cadence: %s", packet.Cadence)
	}
	if len(packet.PhaseOrder) != 5 {
		t.Fatalf("expected 5 phases in packet, got %d", len(packet.PhaseOrder))
	}
	if len(packet.Units) != 5 {
		t.Fatalf("expected 5 persona execution units, got %d", len(packet.Units))
	}

	for i, unit := range packet.Units {
		if unit.PersonaID == "" || unit.DecisionLens == "" || unit.PrimaryQuestion == "" || unit.EscalationTarget == "" {
			t.Fatalf("unit %d has empty required fields: %+v", i, unit)
		}
		if len(unit.RequiredEvidence) == 0 {
			t.Fatalf("persona %s has no evidence requirements", unit.PersonaID)
		}
		if len(unit.PhaseFocus) != len(packet.PhaseOrder) {
			t.Fatalf("persona %s phase focus mismatch: got=%d want=%d", unit.PersonaID, len(unit.PhaseFocus), len(packet.PhaseOrder))
		}
		if len(unit.EntryGateSignals) != len(plan.TriggerSignals) {
			t.Fatalf("persona %s entry gate signals mismatch: got=%d want=%d", unit.PersonaID, len(unit.EntryGateSignals), len(plan.TriggerSignals))
		}
		if i > 0 && packet.Units[i-1].PersonaID > unit.PersonaID {
			t.Fatalf("persona units are not sorted: %q before %q", packet.Units[i-1].PersonaID, unit.PersonaID)
		}
	}
}

func TestBuildPersonaExecutionPacketValidation(t *testing.T) {
	_, err := BuildPersonaExecutionPacket("", DefaultDeepThinkingSwarmPlan())
	if !errors.Is(err, errIssueIDRequired) {
		t.Fatalf("expected issue id required error, got %v", err)
	}

	_, err = BuildPersonaExecutionPacket("sdp_dev-hx0.1.2", DeepThinkingSwarmPlan{})
	if !errors.Is(err, errSwarmPlanIncomplete) {
		t.Fatalf("expected incomplete plan error, got %v", err)
	}
}

func TestAssembleSwarmScoreReportComplete(t *testing.T) {
	packet, err := BuildPersonaExecutionPacket("sdp_dev-hx0.1.2", DefaultDeepThinkingSwarmPlan())
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	report := AssembleSwarmScoreReport(packet, []PersonaScore{
		{PersonaID: "systems-architect", Score: 88, Recommendation: "keep dependency boundaries strict"},
		{PersonaID: "sre", Score: 74, Recommendation: "add rollback drill to weekly runbook"},
		{PersonaID: "security-reviewer", Score: 91, Recommendation: "add abuse-case checklist"},
		{PersonaID: "dx-expert", Score: 95, Recommendation: "publish runnable command transcripts"},
		{PersonaID: "product-strategist", Score: 61, Recommendation: "defer lower-impact roadmap work"},
	})

	if report.PersonaCount != 5 || report.RespondedPersonaCount != 5 {
		t.Fatalf("unexpected persona counts: %+v", report)
	}
	if len(report.MissingPersonaIDs) != 0 {
		t.Fatalf("expected no missing personas, got %v", report.MissingPersonaIDs)
	}
	if len(report.UnknownPersonaIDs) != 0 {
		t.Fatalf("expected no unknown personas, got %v", report.UnknownPersonaIDs)
	}
	if report.AverageScore != 81 {
		t.Fatalf("unexpected average score: %d", report.AverageScore)
	}
	if report.MinScore != 61 || report.MaxScore != 95 {
		t.Fatalf("unexpected score range: min=%d max=%d", report.MinScore, report.MaxScore)
	}
	if !report.ConsensusReached {
		t.Fatalf("expected consensus reached with 4 of 5 personas >=70")
	}
	if len(report.DissentingPersonaIDs) != 1 || report.DissentingPersonaIDs[0] != "product-strategist" {
		t.Fatalf("unexpected dissenting personas: %v", report.DissentingPersonaIDs)
	}
	if len(report.PriorityRecommendations) != 5 {
		t.Fatalf("expected all recommendations preserved, got %v", report.PriorityRecommendations)
	}
	if report.PriorityRecommendations[0] != "dx-expert: publish runnable command transcripts" {
		t.Fatalf("recommendations not sorted by score: %v", report.PriorityRecommendations)
	}
}

func TestAssembleSwarmScoreReportMissingAndUnknown(t *testing.T) {
	packet, err := BuildPersonaExecutionPacket("sdp_dev-hx0.1.2", DefaultDeepThinkingSwarmPlan())
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	report := AssembleSwarmScoreReport(packet, []PersonaScore{
		{PersonaID: "security-reviewer", Score: 120, Recommendation: "enforce strict secret scanning"},
		{PersonaID: "security-reviewer", Score: 65, Recommendation: "duplicate should be ignored"},
		{PersonaID: "systems-architect", Score: 82, Recommendation: "lock dependency policy"},
		{PersonaID: "external-observer", Score: 80, Recommendation: "should be tracked as unknown"},
	})

	if report.RespondedPersonaCount != 2 {
		t.Fatalf("unexpected responded persona count: %d", report.RespondedPersonaCount)
	}
	if report.AverageScore != 91 {
		t.Fatalf("unexpected average for known personas: %d", report.AverageScore)
	}
	if report.MinScore != 82 || report.MaxScore != 100 {
		t.Fatalf("unexpected score clamp behavior: min=%d max=%d", report.MinScore, report.MaxScore)
	}
	if report.ConsensusReached {
		t.Fatalf("did not expect consensus when personas are missing")
	}
	if len(report.MissingPersonaIDs) != 3 {
		t.Fatalf("expected 3 missing personas, got %v", report.MissingPersonaIDs)
	}
	if len(report.UnknownPersonaIDs) != 1 || report.UnknownPersonaIDs[0] != "external-observer" {
		t.Fatalf("unexpected unknown persona ids: %v", report.UnknownPersonaIDs)
	}
}

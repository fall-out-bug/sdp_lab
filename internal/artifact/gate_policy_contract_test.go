package artifact

import (
	"reflect"
	"testing"
)

func TestDefaultGateAggregationModelIsStrict(t *testing.T) {
	model := DefaultGateAggregationModel()
	if model.ContractVersion != GateAggregationContractVersion {
		t.Fatalf("unexpected aggregation contract version: got=%s want=%s", model.ContractVersion, GateAggregationContractVersion)
	}
	if model.Mode != "all-required-signals-pass" {
		t.Fatalf("unexpected aggregation mode: %s", model.Mode)
	}
	if model.UnknownStatusTreatment != "block" {
		t.Fatalf("unexpected unknown status treatment: %s", model.UnknownStatusTreatment)
	}

	if !reflect.DeepEqual(model.PassStatuses, []GateSignalStatus{GateSignalStatusPass}) {
		t.Fatalf("unexpected pass statuses: %#v", model.PassStatuses)
	}
	if !reflect.DeepEqual(model.BlockingStatuses, []GateSignalStatus{GateSignalStatusFail, GateSignalStatusMissing}) {
		t.Fatalf("unexpected blocking statuses: %#v", model.BlockingStatuses)
	}
}

func TestTransitionPolicyContractDerivedFromIntakePrerequisites(t *testing.T) {
	contract := BuildTransitionPolicyContract()
	if contract.ContractVersion != TransitionPolicyContractVersion {
		t.Fatalf("unexpected transition policy contract version: got=%s want=%s", contract.ContractVersion, TransitionPolicyContractVersion)
	}

	prerequisites := TransitionPrerequisites()
	if len(contract.Rules) != len(prerequisites) {
		t.Fatalf("unexpected rule count: got=%d want=%d", len(contract.Rules), len(prerequisites))
	}

	byEdge := map[string]TransitionPrerequisite{}
	for _, prerequisite := range prerequisites {
		edge := prerequisite.FromPhase + "->" + prerequisite.ToPhase
		byEdge[edge] = prerequisite
	}

	seenEdges := map[string]struct{}{}
	for _, rule := range contract.Rules {
		edge := rule.FromPhase + "->" + rule.ToPhase
		prerequisite, ok := byEdge[edge]
		if !ok {
			t.Fatalf("policy rule edge %s not in prerequisites", edge)
		}
		if _, duplicated := seenEdges[edge]; duplicated {
			t.Fatalf("duplicate policy rule edge %s", edge)
		}
		seenEdges[edge] = struct{}{}

		if !reflect.DeepEqual(rule.RequiredGateSignals, prerequisite.RequiredGateSignals) {
			t.Fatalf("gate signal mismatch for %s: got=%#v want=%#v", edge, rule.RequiredGateSignals, prerequisite.RequiredGateSignals)
		}
		if !reflect.DeepEqual(rule.RequiredArtifactClassIDs, prerequisite.RequiredArtifactClassIDs) {
			t.Fatalf("required classes mismatch for %s: got=%#v want=%#v", edge, rule.RequiredArtifactClassIDs, prerequisite.RequiredArtifactClassIDs)
		}
		if !reflect.DeepEqual(rule.RequiredProvenanceKeys, prerequisite.RequiredProvenanceKeys) {
			t.Fatalf("required provenance mismatch for %s: got=%#v want=%#v", edge, rule.RequiredProvenanceKeys, prerequisite.RequiredProvenanceKeys)
		}

		if len(rule.DenialReasonCodes) != 4 {
			t.Fatalf("unexpected denial reason count for %s: got=%d want=4", edge, len(rule.DenialReasonCodes))
		}
		for _, reasonCode := range rule.DenialReasonCodes {
			if reasonCode == "" {
				t.Fatalf("empty denial reason code for %s", edge)
			}
		}
	}

	if len(seenEdges) != len(byEdge) {
		t.Fatalf("not all prerequisite edges mapped to policy rules: got=%d want=%d", len(seenEdges), len(byEdge))
	}
}

func TestTransitionPolicyContractReasonCodesAreDeterministic(t *testing.T) {
	contract := BuildTransitionPolicyContract()
	for _, rule := range contract.Rules {
		want := []string{
			transitionReasonCode(rule.FromPhase, rule.ToPhase, "missing-gate-signal"),
			transitionReasonCode(rule.FromPhase, rule.ToPhase, "gate-not-passed"),
			transitionReasonCode(rule.FromPhase, rule.ToPhase, "missing-artifact"),
			transitionReasonCode(rule.FromPhase, rule.ToPhase, "missing-provenance-key"),
		}
		if !reflect.DeepEqual(rule.DenialReasonCodes, want) {
			t.Fatalf("unexpected reason codes for %s->%s: got=%#v want=%#v", rule.FromPhase, rule.ToPhase, rule.DenialReasonCodes, want)
		}
	}
}

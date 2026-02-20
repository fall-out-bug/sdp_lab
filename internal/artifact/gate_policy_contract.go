package artifact

import "sort"

const (
	GateAggregationContractVersion  = "gate-aggregation/v1"
	TransitionPolicyContractVersion = "transition-policy/v1"
)

type GateSignalStatus string

const (
	GateSignalStatusPass    GateSignalStatus = "pass"
	GateSignalStatusFail    GateSignalStatus = "fail"
	GateSignalStatusMissing GateSignalStatus = "missing"
)

// GateAggregationModel defines how transition gate outcomes are adjudicated.
type GateAggregationModel struct {
	ContractVersion        string
	Mode                   string
	PassStatuses           []GateSignalStatus
	BlockingStatuses       []GateSignalStatus
	UnknownStatusTreatment string
}

// TransitionPolicyRule defines required evidence and gate policy per phase edge.
type TransitionPolicyRule struct {
	FromPhase                string
	ToPhase                  string
	RequiredGateSignals      []string
	RequiredArtifactClassIDs []string
	RequiredProvenanceKeys   []string
	DenialReasonCodes        []string
}

// TransitionPolicyContract combines gate aggregation semantics with phase rules.
type TransitionPolicyContract struct {
	ContractVersion string
	Aggregation     GateAggregationModel
	Rules           []TransitionPolicyRule
}

func DefaultGateAggregationModel() GateAggregationModel {
	return GateAggregationModel{
		ContractVersion:        GateAggregationContractVersion,
		Mode:                   "all-required-signals-pass",
		PassStatuses:           []GateSignalStatus{GateSignalStatusPass},
		BlockingStatuses:       []GateSignalStatus{GateSignalStatusFail, GateSignalStatusMissing},
		UnknownStatusTreatment: "block",
	}
}

func BuildTransitionPolicyContract() TransitionPolicyContract {
	aggregation := DefaultGateAggregationModel()
	prerequisites := TransitionPrerequisites()
	rules := make([]TransitionPolicyRule, 0, len(prerequisites))
	for _, prerequisite := range prerequisites {
		rules = append(rules, TransitionPolicyRule{
			FromPhase:                prerequisite.FromPhase,
			ToPhase:                  prerequisite.ToPhase,
			RequiredGateSignals:      appendCopy(nil, prerequisite.RequiredGateSignals...),
			RequiredArtifactClassIDs: appendCopy(nil, prerequisite.RequiredArtifactClassIDs...),
			RequiredProvenanceKeys:   appendCopy(nil, prerequisite.RequiredProvenanceKeys...),
			DenialReasonCodes: []string{
				transitionReasonCode(prerequisite.FromPhase, prerequisite.ToPhase, "missing-gate-signal"),
				transitionReasonCode(prerequisite.FromPhase, prerequisite.ToPhase, "gate-not-passed"),
				transitionReasonCode(prerequisite.FromPhase, prerequisite.ToPhase, "missing-artifact"),
				transitionReasonCode(prerequisite.FromPhase, prerequisite.ToPhase, "missing-provenance-key"),
			},
		})
	}

	sort.Slice(rules, func(i, j int) bool {
		if rules[i].FromPhase == rules[j].FromPhase {
			return rules[i].ToPhase < rules[j].ToPhase
		}
		return rules[i].FromPhase < rules[j].FromPhase
	})

	return TransitionPolicyContract{
		ContractVersion: TransitionPolicyContractVersion,
		Aggregation: GateAggregationModel{
			ContractVersion:        aggregation.ContractVersion,
			Mode:                   aggregation.Mode,
			PassStatuses:           append([]GateSignalStatus(nil), aggregation.PassStatuses...),
			BlockingStatuses:       append([]GateSignalStatus(nil), aggregation.BlockingStatuses...),
			UnknownStatusTreatment: aggregation.UnknownStatusTreatment,
		},
		Rules: rules,
	}
}

func transitionReasonCode(fromPhase string, toPhase string, reason string) string {
	return "transition-" + fromPhase + "-to-" + toPhase + "-" + reason
}

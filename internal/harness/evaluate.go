package harness

import "fmt"

func EvaluateCompliance(contract *TaskContract, snapshot *TaskSnapshot) ComplianceReport {
	if contract == nil {
		return ComplianceReport{
			GeneratedAt: "",
			Blocked:     true,
			GateResults: []GateResult{{
				GateID: GateRequirementIntegrity,
				Status: GateBlock,
				Violations: []Violation{{
					Type:    DriftScopeWeaken,
					Field:   "contract",
					Message: "contract is required",
				}},
			}},
		}
	}
	if snapshot == nil {
		snapshot = &TaskSnapshot{}
	}

	report := newReport(contract, snapshot)
	req := evaluateRequirementIntegrity(contract, snapshot)
	metric := evaluateMetricParity(contract, snapshot)
	evidence := evaluateEvidenceGate(contract, snapshot)
	quality := evaluateQualityGate(contract, snapshot)
	process := evaluateProcessGate(snapshot)

	report.GateResults = append(report.GateResults, req, evidence, metric, quality, process)
	for _, gate := range report.GateResults {
		if gate.Status == GateBlock {
			report.Blocked = true
			break
		}
	}
	return report
}

func evaluateRequirementIntegrity(contract *TaskContract, snapshot *TaskSnapshot) GateResult {
	res := GateResult{GateID: GateRequirementIntegrity, Status: GatePass}
	actual := make(map[string]struct{}, len(snapshot.AcceptanceCriteria))
	for _, ac := range snapshot.AcceptanceCriteria {
		if ac.ID != "" {
			actual[ac.ID] = struct{}{}
		}
	}
	for _, ac := range contract.AcceptanceCriteria {
		if ac.ID == "" {
			continue
		}
		if _, ok := actual[ac.ID]; !ok {
			res.Status = GateBlock
			res.Violations = append(res.Violations, Violation{
				Type:     DriftACDrop,
				Field:    "acceptance_criteria",
				Message:  fmt.Sprintf("acceptance criterion %q is missing", ac.ID),
				Expected: ac.ID,
			})
		}
	}
	if !contract.Constraints.AllowScopeReduction {
		if len(snapshot.AcceptanceCriteria) < len(contract.AcceptanceCriteria) {
			res.Status = GateBlock
			res.Violations = append(res.Violations, Violation{
				Type:     DriftScopeWeaken,
				Field:    "acceptance_criteria_count",
				Message:  "acceptance criteria count decreased",
				Expected: fmt.Sprintf("%d", len(contract.AcceptanceCriteria)),
				Actual:   fmt.Sprintf("%d", len(snapshot.AcceptanceCriteria)),
			})
		}
	}
	return res
}

func evaluateMetricParity(contract *TaskContract, snapshot *TaskSnapshot) GateResult {
	res := GateResult{GateID: GateMetricParity, Status: GatePass}
	actual := make(map[string]struct{}, len(snapshot.Metrics))
	for _, metric := range snapshot.Metrics {
		if metric.Name != "" {
			actual[metric.Name] = struct{}{}
		}
	}
	for _, metric := range contract.RequiredMetrics {
		if metric.Name == "" {
			continue
		}
		if _, ok := actual[metric.Name]; !ok {
			res.Status = GateBlock
			res.Violations = append(res.Violations, Violation{
				Type:     DriftMetricDrop,
				Field:    "required_metrics",
				Message:  fmt.Sprintf("required metric %q is missing", metric.Name),
				Expected: metric.Name,
			})
		}
	}
	if !contract.Constraints.AllowMetricReduction {
		if len(snapshot.Metrics) < len(contract.RequiredMetrics) {
			res.Status = GateBlock
			res.Violations = append(res.Violations, Violation{
				Type:     DriftMetricDrop,
				Field:    "metrics_count",
				Message:  "metrics set size decreased",
				Expected: fmt.Sprintf("%d", len(contract.RequiredMetrics)),
				Actual:   fmt.Sprintf("%d", len(snapshot.Metrics)),
			})
		}
	}
	return res
}

func evaluateEvidenceGate(contract *TaskContract, snapshot *TaskSnapshot) GateResult {
	res := GateResult{GateID: GateEvidence, Status: GatePass}
	actualEvidence := make(map[string]struct{}, len(snapshot.Evidence))
	for _, ev := range snapshot.Evidence {
		if ev != "" {
			actualEvidence[ev] = struct{}{}
		}
	}

	for _, ev := range contract.RequiredEvidence {
		if _, ok := actualEvidence[ev]; !ok {
			res.Status = GateBlock
			res.Violations = append(res.Violations, Violation{
				Type:     DriftMissingEvidence,
				Field:    "required_evidence",
				Message:  fmt.Sprintf("required evidence %q is missing", ev),
				Expected: ev,
			})
		}
	}

	for _, claim := range snapshot.Claims {
		if len(claim.EvidenceRefs) == 0 {
			res.Status = GateBlock
			res.Violations = append(res.Violations, Violation{
				Type:    DriftUnsupportedClaim,
				Field:   "claims",
				Message: fmt.Sprintf("claim %q has no evidence references", claim.ID),
			})
			continue
		}
		for _, ref := range claim.EvidenceRefs {
			if _, ok := actualEvidence[ref]; !ok {
				res.Status = GateBlock
				res.Violations = append(res.Violations, Violation{
					Type:     DriftUnsupportedClaim,
					Field:    "claims.evidence_refs",
					Message:  fmt.Sprintf("claim %q references missing evidence %q", claim.ID, ref),
					Expected: ref,
				})
			}
		}
	}

	return res
}

func evaluateQualityGate(contract *TaskContract, snapshot *TaskSnapshot) GateResult {
	res := GateResult{GateID: GateQuality, Status: GatePass}
	checks := map[string]bool{
		"build":     contract.QualityGates.Build,
		"test":      contract.QualityGates.Test,
		"lint":      contract.QualityGates.Lint,
		"typecheck": contract.QualityGates.Typecheck,
	}
	for check, required := range checks {
		if !required {
			continue
		}
		if !snapshot.QualityResults[check] {
			res.Status = GateBlock
			res.Violations = append(res.Violations, Violation{
				Type:     DriftQualityGateFail,
				Field:    "quality_results." + check,
				Message:  fmt.Sprintf("quality gate %q failed or missing", check),
				Expected: "true",
				Actual:   "false",
			})
		}
	}
	return res
}

func evaluateProcessGate(snapshot *TaskSnapshot) GateResult {
	res := GateResult{GateID: GateProcess, Status: GatePass}
	required := map[string]bool{
		"contract_coverage_summary": snapshot.ProcessReport.ContractCoverageSummary,
		"gate_results":              snapshot.ProcessReport.GateResults,
		"evidence_index":            snapshot.ProcessReport.EvidenceIndex,
		"decision_log":              snapshot.ProcessReport.DecisionLog,
	}
	for field, ok := range required {
		if ok {
			continue
		}
		res.Status = GateBlock
		res.Violations = append(res.Violations, Violation{
			Type:     DriftProcessIncomplete,
			Field:    field,
			Message:  fmt.Sprintf("process report field %q is required", field),
			Expected: "true",
			Actual:   "false",
		})
	}
	return res
}

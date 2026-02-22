package review

// Consensus computes consensus from N verdicts.
func Consensus(verdicts []ReviewVerdict) ConsensusResult {
	result := ConsensusResult{Verdicts: verdicts}
	if len(verdicts) == 0 {
		result.Consensus = "needs_changes"
		return result
	}

	approveCount := 0
	rejectCount := 0
	var dissenting []string

	for _, v := range verdicts {
		switch v.Verdict {
		case "approve":
			approveCount++
		case "reject":
			rejectCount++
			dissenting = append(dissenting, v.PersonaID)
		case "needs_changes":
			dissenting = append(dissenting, v.PersonaID)
		}
	}

	result.Dissenting = dissenting
	threshold := (len(verdicts) + 1) / 2

	if rejectCount > 0 {
		result.Consensus = "reject"
		result.Rejected = true
		return result
	}
	if approveCount >= threshold {
		result.Consensus = "approve"
		result.Approved = true
		return result
	}
	result.Consensus = "needs_changes"
	result.NeedsChanges = true
	return result
}

package review

// AggregateFeedback builds a ReworkRequest from verdicts that need changes.
func AggregateFeedback(projectID, issueID, runID string, verdicts []ReviewVerdict) ReworkRequest {
	req := ReworkRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		RunID:     runID,
	}
	seen := make(map[string]bool)
	for _, v := range verdicts {
		if v.Verdict == "needs_changes" || v.Verdict == "reject" {
			for _, c := range v.Comments {
				if c != "" && !seen[c] {
					seen[c] = true
					req.Comments = append(req.Comments, c)
				}
			}
			if v.PersonaID != "" && !seen[v.PersonaID] {
				seen[v.PersonaID] = true
				req.Personas = append(req.Personas, v.PersonaID)
			}
		}
	}
	return req
}

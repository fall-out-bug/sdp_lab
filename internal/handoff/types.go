package handoff

// AnalystHandoff is the structured output from the analyst role.
type AnalystHandoff struct {
	RiskClass          string   `json:"risk_class"`
	DecomposedSteps    []string `json:"decomposed_steps"`
	RecommendedApproach string `json:"recommended_approach"`
	EstimatedComplexity string `json:"estimated_complexity"`
	ScopeFiles         []string `json:"scope_files"`
}

// CoderHandoff is the structured output from the coder role.
type CoderHandoff struct {
	ChangedFiles       []string     `json:"changed_files"`
	TestResults        TestResults  `json:"test_results"`
	ImplementationNotes string     `json:"implementation_notes"`
	Branch             string      `json:"branch"`
	Commits            []string    `json:"commits"`
}

// TestResults holds test execution metrics.
type TestResults struct {
	Passed   int     `json:"passed"`
	Failed   int     `json:"failed"`
	Coverage float64 `json:"coverage"`
}

// ReviewerHandoff is the structured output from the reviewer role.
type ReviewerHandoff struct {
	Verdict        string   `json:"verdict"`
	Findings       []string `json:"findings"`
	Suggestions    []string `json:"suggestions"`
	RiskAssessment string   `json:"risk_assessment"`
}

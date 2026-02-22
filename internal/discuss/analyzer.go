package discuss

import "context"

// Analyzer analyzes a feature idea and returns structured analysis (scope, risks, subtasks).
type Analyzer interface {
	Analyze(ctx context.Context, sess *Session) (*AnalysisResult, error)
}

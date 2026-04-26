package wsverdict

// TestReport contains test run summary.
type TestReport struct {
	Failed   int
	Errored  int
	Skipped  int
	Total    int
	Coverage float64 // 0..100
}

// GuardDiff contains guard check results.
type GuardDiff struct {
	OutOfScope []string // files modified outside allowed paths
}

// WsVerdictInput is the input to WsVerdictMicro Stage.
type WsVerdictInput struct {
	Report        TestReport
	Guard         GuardDiff
	MinCoverage   float64 // minimum required coverage (0 = no requirement)
	SkipThreshold int     // max allowed skipped tests (0 = no limit)
}

// RulesConfig holds thresholds (with sensible defaults via Default()).
type RulesConfig struct {
	SkipThreshold int     // default 5
	MinCoverage   float64 // default 0 (disabled)
}

// Default returns a RulesConfig with sensible defaults.
func Default() RulesConfig {
	return RulesConfig{SkipThreshold: 5}
}

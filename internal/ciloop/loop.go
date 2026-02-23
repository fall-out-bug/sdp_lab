package ciloop

import "time"

// LoopResult is the outcome of RunLoop.
type LoopResult int

const (
	ResultGreen     LoopResult = iota // all checks passed
	ResultEscalated                   // escalation triggered
	ResultMaxIter                     // max iterations exceeded
)

// Fixer attempts to fix a set of auto-fixable failing checks.
// Returns an error if the fix cannot be applied.
type Fixer interface {
	Fix(checks []CheckResult) error
}

// LoopOptions configures RunLoop behaviour.
type LoopOptions struct {
	PRNumber   int
	MaxIter    int
	PollDelay  time.Duration
	RetryDelay time.Duration
	Poller     *Poller
	// OnEscalate is called when a non-auto-fixable failure is detected.
	OnEscalate func(checks []CheckResult) error
	// Fixer handles auto-fixable failures. Nil means escalate auto-fixable too.
	Fixer Fixer
}

// RunLoop polls CI checks until green, escalation, or max iterations.
//
// PENDING checks trigger a RetryDelay wait without consuming an iteration.
// FAILURE checks are classified; unfixable ones call OnEscalate immediately.
// Auto-fixable failures increment iteration count and call Fixer when set.
// Exit criteria:
//   - ResultGreen     when IsAllGreen
//   - ResultEscalated when OnEscalate is called
//   - ResultMaxIter   when iter >= MaxIter
func RunLoop(opts LoopOptions) (LoopResult, error) {
	iter := 0
	for {
		if opts.PollDelay > 0 {
			time.Sleep(opts.PollDelay)
		}

		checks, err := opts.Poller.GetChecks(opts.PRNumber)
		if err != nil {
			return ResultEscalated, err
		}

		if IsAllGreen(checks) {
			return ResultGreen, nil
		}

		pending := FilterByState(checks, StatePending)
		inProgress := FilterByState(checks, StateInProgress)
		if len(pending)+len(inProgress) > 0 {
			if opts.RetryDelay > 0 {
				time.Sleep(opts.RetryDelay)
			}
			continue
		}

		failing := append(FilterByState(checks, StateFailure), FilterByState(checks, StateError)...)
		if len(failing) == 0 {
			// No pending, no failures — treat as green.
			return ResultGreen, nil
		}

		escalateChecks := make([]CheckResult, 0)
		autoFixChecks := make([]CheckResult, 0)
		for _, c := range failing {
			if Classify(c.Name) == ClassAutoFixable {
				autoFixChecks = append(autoFixChecks, c)
			} else {
				escalateChecks = append(escalateChecks, c)
			}
		}

		// Non-auto-fixable failures trigger immediate escalation.
		if len(escalateChecks) > 0 {
			if opts.OnEscalate != nil {
				if err := opts.OnEscalate(escalateChecks); err != nil {
					return ResultEscalated, err
				}
			}
			return ResultEscalated, nil
		}

		// Auto-fixable failures: count iteration and attempt fix if Fixer present.
		iter++
		if iter >= opts.MaxIter {
			return ResultMaxIter, nil
		}

		if opts.Fixer != nil {
			if err := opts.Fixer.Fix(autoFixChecks); err != nil {
				if opts.OnEscalate != nil {
					opts.OnEscalate(autoFixChecks)
				}
				return ResultEscalated, err
			}
		}
	}
}

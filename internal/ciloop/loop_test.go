package ciloop_test

import (
	"testing"
	"time"

	"sdp_dev/internal/ciloop"
)

// loopRunner simulates sequences of gh responses across calls.
type sequenceRunner struct {
	responses [][]byte
	errs      []error
	call      int
}

func (s *sequenceRunner) Run(_ string, _ ...string) ([]byte, error) {
	i := s.call
	if i >= len(s.responses) {
		i = len(s.responses) - 1
	}
	s.call++
	return s.responses[i], s.errs[i]
}

func newSequence(responses [][]byte) *sequenceRunner {
	errs := make([]error, len(responses))
	return &sequenceRunner{responses: responses, errs: errs}
}

func TestRunLoopGreenFirstTry(t *testing.T) {
	runner := newSequence([][]byte{greenJSON})
	opts := ciloop.LoopOptions{
		PRNumber:    42,
		MaxIter:     5,
		PollDelay:   0,
		RetryDelay:  0,
		Poller:      ciloop.NewPoller(runner),
		OnEscalate:  func(checks []ciloop.CheckResult) error { return nil },
	}
	result, err := ciloop.RunLoop(opts)
	if err != nil {
		t.Fatal(err)
	}
	if result != ciloop.ResultGreen {
		t.Errorf("expected Green, got %v", result)
	}
}

func TestRunLoopPendingThenGreen(t *testing.T) {
	runner := newSequence([][]byte{pendingJSON, greenJSON})
	opts := ciloop.LoopOptions{
		PRNumber:   42,
		MaxIter:    5,
		PollDelay:  0,
		RetryDelay: 0,
		Poller:     ciloop.NewPoller(runner),
		OnEscalate: func(checks []ciloop.CheckResult) error { return nil },
	}
	result, err := ciloop.RunLoop(opts)
	if err != nil {
		t.Fatal(err)
	}
	if result != ciloop.ResultGreen {
		t.Errorf("expected Green, got %v", result)
	}
}

func TestRunLoopEscalatesOnUnfixableFailure(t *testing.T) {
	secretsFailure := []byte(`[{"name":"secrets-scan","state":"FAILURE"}]`)
	runner := newSequence([][]byte{secretsFailure})
	escalated := false
	opts := ciloop.LoopOptions{
		PRNumber:   42,
		MaxIter:    5,
		PollDelay:  0,
		RetryDelay: 0,
		Poller:     ciloop.NewPoller(runner),
		OnEscalate: func(checks []ciloop.CheckResult) error {
			escalated = true
			return nil
		},
	}
	result, err := ciloop.RunLoop(opts)
	if err != nil {
		t.Fatal(err)
	}
	if result != ciloop.ResultEscalated {
		t.Errorf("expected Escalated, got %v", result)
	}
	if !escalated {
		t.Error("OnEscalate was not called")
	}
}

func TestRunLoopExceedsMaxIter(t *testing.T) {
	goTestFailure := []byte(`[{"name":"go-test","state":"FAILURE"}]`)
	responses := make([][]byte, 10)
	for i := range responses {
		responses[i] = goTestFailure
	}
	runner := newSequence(responses)
	opts := ciloop.LoopOptions{
		PRNumber:   42,
		MaxIter:    3,
		PollDelay:  0,
		RetryDelay: 0,
		Poller:     ciloop.NewPoller(runner),
		OnEscalate: func(checks []ciloop.CheckResult) error { return nil },
		// No fixer = auto-fixable checks treated as escalate when no fixer
		Fixer: nil,
	}
	result, err := ciloop.RunLoop(opts)
	if err != nil {
		t.Fatal(err)
	}
	if result != ciloop.ResultMaxIter {
		t.Errorf("expected MaxIter, got %v", result)
	}
}

func TestLoopOptionsPollDelayIsRespected(t *testing.T) {
	runner := newSequence([][]byte{greenJSON})
	start := time.Now()
	opts := ciloop.LoopOptions{
		PRNumber:   42,
		MaxIter:    5,
		PollDelay:  10 * time.Millisecond,
		RetryDelay: 0,
		Poller:     ciloop.NewPoller(runner),
		OnEscalate: func(checks []ciloop.CheckResult) error { return nil },
	}
	ciloop.RunLoop(opts)
	elapsed := time.Since(start)
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected poll delay of at least 10ms, elapsed: %v", elapsed)
	}
}

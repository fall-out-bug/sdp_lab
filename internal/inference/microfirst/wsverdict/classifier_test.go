package wsverdict

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"sdp_dev/internal/inference/decompose"
)

// fixtureCase is the JSON shape for testdata/fixtures.json.
type fixtureCase struct {
	Name  string `json:"name"`
	Input struct {
		Report struct {
			Failed   int     `json:"failed"`
			Errored  int     `json:"errored"`
			Skipped  int     `json:"skipped"`
			Total    int     `json:"total"`
			Coverage float64 `json:"coverage"`
		} `json:"report"`
		Guard struct {
			OutOfScope []string `json:"out_of_scope"`
		} `json:"guard"`
		MinCoverage   float64 `json:"min_coverage"`
		SkipThreshold int     `json:"skip_threshold"`
	} `json:"input"`
	Expected struct {
		Verdict    string  `json:"verdict"`
		Confidence float64 `json:"confidence"`
		Status     string  `json:"status"`
	} `json:"expected"`
}

func TestWsVerdictMicro_Fixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures.json")
	if err != nil {
		t.Fatalf("failed to read fixtures: %v", err)
	}
	var cases []fixtureCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed to unmarshal fixtures: %v", err)
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			cfg := RulesConfig{
				SkipThreshold: tc.Input.SkipThreshold,
				MinCoverage:   tc.Input.MinCoverage,
			}
			micro := New(cfg)
			in := WsVerdictInput{
				Report: TestReport{
					Failed:   tc.Input.Report.Failed,
					Errored:  tc.Input.Report.Errored,
					Skipped:  tc.Input.Report.Skipped,
					Total:    tc.Input.Report.Total,
					Coverage: tc.Input.Report.Coverage,
				},
				Guard: GuardDiff{
					OutOfScope: tc.Input.Guard.OutOfScope,
				},
				MinCoverage:   tc.Input.MinCoverage,
				SkipThreshold: tc.Input.SkipThreshold,
			}

			res, trace, err := micro.Run(context.Background(), in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if trace.Attempts != 1 {
				t.Errorf("trace.Attempts = %d, want 1", trace.Attempts)
			}
			if string(res.Verdict) != tc.Expected.Verdict {
				t.Errorf("verdict = %q, want %q", res.Verdict, tc.Expected.Verdict)
			}
			if res.Confidence() != tc.Expected.Confidence {
				t.Errorf("confidence = %v, want %v", res.Confidence(), tc.Expected.Confidence)
			}
			if string(res.ConfStatus()) != tc.Expected.Status {
				t.Errorf("status = %q, want %q", res.ConfStatus(), tc.Expected.Status)
			}
		})
	}
}

// --- Inline tests ---

func newMicro() *WsVerdictMicro { return New(Default()) }

func TestR1_FailedGreaterZero(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Failed: 1, Total: 10, Coverage: 90},
	})
	assertVerdict(t, res, VerdictFail, 0.95, decompose.StatusOK)
}

func TestR1_FailedMany(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Failed: 10, Total: 20, Coverage: 50},
	})
	assertVerdict(t, res, VerdictFail, 0.95, decompose.StatusOK)
}

func TestR1_TakesPrecedenceOverOutOfScope(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Failed: 1, Total: 5},
		Guard:  GuardDiff{OutOfScope: []string{"x.go"}},
	})
	assertVerdict(t, res, VerdictFail, 0.95, decompose.StatusOK)
}

func TestR2_CleanPassNoSkips(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Total: 30, Coverage: 100},
	})
	assertVerdict(t, res, VerdictPass, 0.90, decompose.StatusOK)
}

func TestR2_PassWithSkipsAtThreshold(t *testing.T) {
	// Skipped == threshold (5) — not over, so still PASS via R2
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Skipped: 5, Total: 20, Coverage: 80},
	})
	assertVerdict(t, res, VerdictPass, 0.90, decompose.StatusOK)
}

func TestR2_PassIgnoresMinCoverageWhenZero(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report:      TestReport{Total: 10, Coverage: 10},
		MinCoverage: 0,
	})
	assertVerdict(t, res, VerdictPass, 0.90, decompose.StatusOK)
}

func TestR3_OutOfScopeWithErrored(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Errored: 1, Total: 5},
		Guard:  GuardDiff{OutOfScope: []string{"vendor/lib.go"}},
	})
	assertVerdict(t, res, VerdictUnsure, 0.40, decompose.StatusUnsure)
}

func TestR3_OutOfScopeNoErrors(t *testing.T) {
	// Errored==0 but out-of-scope present: R2 fails (len(OutOfScope)>0), then R3 fires
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Total: 10, Coverage: 80},
		Guard:  GuardDiff{OutOfScope: []string{"docs/CHANGELOG.md"}},
	})
	assertVerdict(t, res, VerdictUnsure, 0.40, decompose.StatusUnsure)
}

func TestR4_SkippedOverThreshold(t *testing.T) {
	// Errored=1 so R2 is skipped; no OutOfScope so R3 skipped; skipped>5 → R4
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Errored: 1, Skipped: 6, Total: 20},
	})
	assertVerdict(t, res, VerdictUnsure, 0.50, decompose.StatusUnsure)
}

func TestR4_SkippedAtThresholdPlusOne(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Errored: 1, Skipped: 6, Total: 10},
	})
	assertVerdict(t, res, VerdictUnsure, 0.50, decompose.StatusUnsure)
}

func TestR4_SkippedAtThresholdNotOver(t *testing.T) {
	// Skipped==threshold → R4 does not fire; falls to R6 (errored=1, no out-of-scope)
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Errored: 1, Skipped: 5, Total: 10},
	})
	assertVerdict(t, res, VerdictUnsure, 0.30, decompose.StatusUnsure)
}

func TestR4_CustomSkipThreshold(t *testing.T) {
	m := New(RulesConfig{SkipThreshold: 2})
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Errored: 1, Skipped: 3, Total: 10},
	})
	assertVerdict(t, res, VerdictUnsure, 0.50, decompose.StatusUnsure)
}

func TestR5_CoverageBelowMin(t *testing.T) {
	// Errored=1 → R2 skip; no OutOfScope → R3 skip; Skipped<=threshold → R4 skip; R5 fires
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report:      TestReport{Errored: 1, Total: 10, Coverage: 75},
		MinCoverage: 80,
	})
	assertVerdict(t, res, VerdictUnsure, 0.45, decompose.StatusUnsure)
}

func TestR5_CoverageExactlyAtMin(t *testing.T) {
	// Coverage == MinCoverage → R5 does NOT fire (< not <=); falls to R6
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report:      TestReport{Errored: 1, Total: 10, Coverage: 80},
		MinCoverage: 80,
	})
	assertVerdict(t, res, VerdictUnsure, 0.30, decompose.StatusUnsure)
}

func TestR5_MinCoverageZeroDisabled(t *testing.T) {
	// MinCoverage=0 → R5 disabled even if coverage is 0
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report:      TestReport{Errored: 1, Total: 10, Coverage: 0},
		MinCoverage: 0,
	})
	// Falls to R6
	assertVerdict(t, res, VerdictUnsure, 0.30, decompose.StatusUnsure)
}

func TestR6_CatchAll(t *testing.T) {
	// Errored=1, no out-of-scope, skipped<=threshold, no coverage requirement
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Errored: 1, Total: 5, Coverage: 90},
	})
	assertVerdict(t, res, VerdictUnsure, 0.30, decompose.StatusUnsure)
}

// --- Confider interface compliance ---

func TestConfider_PassImplementsInterface(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Total: 10, Coverage: 100},
	})
	var _ decompose.Confider = res
	if res.Confidence() < 0.85 {
		t.Errorf("PASS confidence should be >= 0.85, got %v", res.Confidence())
	}
	if res.ConfStatus() != decompose.StatusOK {
		t.Errorf("PASS status should be StatusOK, got %q", res.ConfStatus())
	}
}

func TestConfider_FailImplementsInterface(t *testing.T) {
	m := newMicro()
	res, _, _ := m.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Failed: 1, Total: 10},
	})
	var _ decompose.Confider = res
	if res.Confidence() < 0.85 {
		t.Errorf("FAIL confidence should be >= 0.85, got %v", res.Confidence())
	}
	if res.ConfStatus() != decompose.StatusOK {
		t.Errorf("FAIL status should be StatusOK, got %q", res.ConfStatus())
	}
}

func TestConfider_UnsureHasLowConfidence(t *testing.T) {
	cases := []WsVerdictInput{
		// R3
		{Report: TestReport{Errored: 1}, Guard: GuardDiff{OutOfScope: []string{"x.go"}}},
		// R4
		{Report: TestReport{Errored: 1, Skipped: 6}},
		// R5
		{Report: TestReport{Errored: 1, Coverage: 70}, MinCoverage: 80},
		// R6
		{Report: TestReport{Errored: 1}},
	}
	m := newMicro()
	for i, in := range cases {
		res, _, _ := m.Run(context.Background(), in)
		if res.Verdict != VerdictUnsure {
			t.Errorf("case %d: expected VerdictUnsure, got %v", i, res.Verdict)
		}
		if res.Confidence() >= 0.85 {
			t.Errorf("case %d: UNSURE confidence should be < 0.85, got %v", i, res.Confidence())
		}
	}
}

// --- WithEscalation integration check ---

func TestWithEscalation_PassShortCircuits(t *testing.T) {
	micro := New(Default())
	llmCalled := false
	llm := decompose.NewStage[WsVerdictInput, WsVerdictMicroResult](
		"fake-llm",
		func(_ context.Context, _ WsVerdictInput) (WsVerdictMicroResult, decompose.StageTrace, error) {
			llmCalled = true
			return WsVerdictMicroResult{}, decompose.StageTrace{}, nil
		},
	)

	combined := decompose.WithEscalation[WsVerdictInput, WsVerdictMicroResult](
		micro, llm,
		decompose.EscalationConfig{ConfidenceThreshold: 0.85},
	)

	res, _, err := combined.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Total: 20, Coverage: 100},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llmCalled {
		t.Error("LLM should not be called when micro is confident PASS")
	}
	if res.Verdict != VerdictPass {
		t.Errorf("expected VerdictPass, got %v", res.Verdict)
	}
}

func TestWithEscalation_UnsureEscalates(t *testing.T) {
	micro := New(Default())
	llmCalled := false
	llm := decompose.NewStage[WsVerdictInput, WsVerdictMicroResult](
		"fake-llm",
		func(_ context.Context, _ WsVerdictInput) (WsVerdictMicroResult, decompose.StageTrace, error) {
			llmCalled = true
			return WsVerdictMicroResult{Verdict: VerdictPass}, decompose.StageTrace{Attempts: 1}, nil
		},
	)

	combined := decompose.WithEscalation[WsVerdictInput, WsVerdictMicroResult](
		micro, llm,
		decompose.EscalationConfig{ConfidenceThreshold: 0.85},
	)

	// R3 → UNSURE, confidence 0.40 < 0.85 → should escalate
	_, _, err := combined.Run(context.Background(), WsVerdictInput{
		Report: TestReport{Errored: 1},
		Guard:  GuardDiff{OutOfScope: []string{"x.go"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !llmCalled {
		t.Error("LLM should be called when micro is unsure")
	}
}

// --- Latency p99 < 5ms ---

func TestLatency_P99Under5ms(t *testing.T) {
	m := newMicro()
	in := WsVerdictInput{
		Report: TestReport{Failed: 0, Errored: 0, Skipped: 2, Total: 100, Coverage: 85},
		Guard:  GuardDiff{},
	}

	const iterations = 1000
	latencies := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		m.classify(in)
		latencies[i] = time.Since(start)
	}

	// sort to find p99
	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[j] < latencies[i] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}
	p99 := latencies[int(0.99*float64(iterations))]
	if p99 >= 5*time.Millisecond {
		t.Errorf("p99 latency %v >= 5ms", p99)
	}
}

// --- Reasons field populated ---

func TestReasons_NotEmpty(t *testing.T) {
	m := newMicro()
	cases := []WsVerdictInput{
		{Report: TestReport{Failed: 1}},
		{Report: TestReport{Total: 10, Coverage: 100}},
		{Report: TestReport{Errored: 1}, Guard: GuardDiff{OutOfScope: []string{"x.go"}}},
		{Report: TestReport{Errored: 1, Skipped: 6}},
		{Report: TestReport{Errored: 1, Coverage: 70}, MinCoverage: 80},
		{Report: TestReport{Errored: 1}},
	}
	for i, in := range cases {
		res, _, _ := m.Run(context.Background(), in)
		if len(res.Reasons) == 0 {
			t.Errorf("case %d: expected non-empty Reasons", i)
		}
	}
}

// --- Name ---

func TestName(t *testing.T) {
	m := newMicro()
	if m.Name() != "wsverdict-micro" {
		t.Errorf("Name() = %q, want %q", m.Name(), "wsverdict-micro")
	}
}

// assertVerdict is a helper for concise assertions.
func assertVerdict(t *testing.T, res WsVerdictMicroResult, wantVerdict Verdict, wantConf float64, wantStatus decompose.Status) {
	t.Helper()
	if res.Verdict != wantVerdict {
		t.Errorf("verdict = %q, want %q", res.Verdict, wantVerdict)
	}
	if res.Confidence() != wantConf {
		t.Errorf("confidence = %v, want %v", res.Confidence(), wantConf)
	}
	if res.ConfStatus() != wantStatus {
		t.Errorf("status = %q, want %q", res.ConfStatus(), wantStatus)
	}
}

package wsverdict_test

import (
	"context"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose/adapters/wsverdict"
	wsverdict_micro "github.com/fall-out-bug/sdp_lab/internal/inference/microfirst/wsverdict"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// diffWithFAIL contains explicit test failure markers that diffToMicroInput will parse.
var diffWithFAIL = wsverdict.Diff{
	WSID:     "F147-test-fail",
	DiffText: "--- FAIL\tTestFoo (0.01s)\nFAIL\tsdp_dev/internal/foo\n",
	Context:  "workstream with failing tests",
}

// diffCleanPass has no failure markers and no skip markers.
var diffCleanPass = wsverdict.Diff{
	WSID:     "F147-test-pass",
	DiffText: "ok  \tsdp_dev/internal/bar\t0.01s\ncoverage: 85.0% of statements\n",
	Context:  "workstream with passing tests",
}

// diffAmbiguous has an error signal (build failed) which sets Errored>0 preventing R2 PASS
// but no test failures — micro returns UNSURE catch-all, LLM pipeline falls through.
var diffAmbiguous = wsverdict.Diff{
	WSID:     "F147-test-ambig",
	DiffText: "ok  \tsdp_dev/internal/baz\t0.01s\nbuild failed: missing dependency\n",
	Context:  "workstream with build error but no test failures",
}

// =====================================================
// MicroFirst pre-gate tests (F147-07)
// =====================================================

// MicroFirst=true + FAIL diff → micro returns confident FAIL, LLM not called.
func TestMicroFirst_FailDiff_NoLLM(t *testing.T) {
	mock := &mockLLM{responses: []string{
		// Provide LLM responses just in case, but they must NOT be called.
		extractClean,
		"failed",
		aggFailed,
	}}

	runner := wsverdict.NewDecomposedRunnerWithOpts(mock, wsverdict.DecomposedRunnerOptions{
		MicroFirst: true,
		MicroRules: wsverdict_micro.RulesConfig{},
	})
	res, err := runner.Run(context.Background(), diffWithFAIL)
	require.NoError(t, err)
	assert.Equal(t, "failed", res.Answer.Verdict)
	assert.Equal(t, 0, mock.calls, "LLM should NOT be called when micro gives confident FAIL")
}

// MicroFirst=true + clean PASS diff (no FAIL, no SKIP) → micro returns confident PASS, LLM not called.
func TestMicroFirst_PassDiff_NoLLM(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractClean,
		"passed",
		aggPassed,
	}}

	runner := wsverdict.NewDecomposedRunnerWithOpts(mock, wsverdict.DecomposedRunnerOptions{
		MicroFirst: true,
		MicroRules: wsverdict_micro.RulesConfig{},
	})
	res, err := runner.Run(context.Background(), diffCleanPass)
	require.NoError(t, err)
	assert.Equal(t, "passed", res.Answer.Verdict)
	assert.Equal(t, 0, mock.calls, "LLM should NOT be called when micro gives confident PASS")
}

// MicroFirst=true + ambiguous diff (many SKIPs) → micro returns UNSURE, pipeline.Run called.
func TestMicroFirst_AmbiguousDiff_LLMCalled(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractClean,
		"passed",
		aggPassed,
	}}

	runner := wsverdict.NewDecomposedRunnerWithOpts(mock, wsverdict.DecomposedRunnerOptions{
		MicroFirst: true,
		MicroRules: wsverdict_micro.RulesConfig{},
	})
	res, err := runner.Run(context.Background(), diffAmbiguous)
	require.NoError(t, err)
	assert.Equal(t, "passed", res.Answer.Verdict)
	assert.Equal(t, 3, mock.calls, "LLM pipeline should be called when micro returns UNSURE")
}

// MicroFirst=false → LLM pipeline is always called, even for an obvious FAIL diff.
func TestMicroFirst_Disabled_LLMAlwaysCalled(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractFail,
		"failed",
		aggFailed,
	}}

	runner := wsverdict.NewDecomposedRunnerWithOpts(mock, wsverdict.DecomposedRunnerOptions{
		MicroFirst: false,
	})
	res, err := runner.Run(context.Background(), diffWithFAIL)
	require.NoError(t, err)
	assert.Equal(t, "failed", res.Answer.Verdict)
	assert.Equal(t, 3, mock.calls, "LLM pipeline must be called when MicroFirst=false")
}

// NewDecomposedRunner is an alias with MicroFirst=true (regression test).
func TestNewDecomposedRunner_IsMicroFirstDefault(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractClean,
		"passed",
		aggPassed,
	}}

	// Using the default constructor — micro gate should fire for a FAIL diff.
	mockFail := &mockLLM{responses: []string{
		extractFail,
		"failed",
		aggFailed,
	}}

	runner := wsverdict.NewDecomposedRunner(mockFail)
	res, err := runner.Run(context.Background(), diffWithFAIL)
	require.NoError(t, err)
	// MicroFirst=true by default → LLM not called for confident FAIL.
	assert.Equal(t, "failed", res.Answer.Verdict)
	assert.Equal(t, 0, mockFail.calls, "default NewDecomposedRunner should have MicroFirst=true")

	// Silence unused variable lint.
	_ = mock
}

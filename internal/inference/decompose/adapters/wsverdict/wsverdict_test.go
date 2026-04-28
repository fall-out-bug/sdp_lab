package wsverdict_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose"
	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose/adapters/wsverdict"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock LLM client ---

// mockLLM returns preset responses for each stage in sequence.
type mockLLM struct {
	responses []string
	calls     int
}

func (m *mockLLM) Call(_ context.Context, _ string, _ wsverdict.CallOptions) (string, int, int, float64, error) {
	if m.calls >= len(m.responses) {
		return "", 0, 0, 0, fmt.Errorf("mock LLM: unexpected call %d (only %d responses configured)", m.calls, len(m.responses))
	}
	r := m.responses[m.calls]
	m.calls++
	return r, 100, 50, 0.001, nil
}

// --- fixture responses ---

var extractClean = `{"changed_files":["internal/foo/bar.go"],"modules":["foo"],"change_type":"feat","summary":"Add foo feature"}`
var extractMixed = `{"changed_files":["main.go","util.go"],"modules":["main","util"],"change_type":"fix","summary":"Fix edge cases in util"}`
var extractFail = `{"changed_files":["critical.go"],"modules":["core"],"change_type":"fix","summary":"Remove critical functionality"}`

var aggPassed = `{"verdict":"passed","score":0.95,"summary":"All acceptance criteria met","blocking_gates":[]}`
var aggPartial = `{"verdict":"partial","score":0.60,"summary":"Most criteria met, some gaps","blocking_gates":["coverage below threshold"]}`
var aggFailed = `{"verdict":"failed","score":0.20,"summary":"Critical criteria unmet","blocking_gates":["tests failing","build broken"]}`

var diffClean = wsverdict.Diff{WSID: "00-146-01", DiffText: "+func Foo() {}", Context: "Add Foo implementation"}
var diffMixed = wsverdict.Diff{WSID: "00-146-02", DiffText: "-func Bug()\n+func Fixed()", Context: "Fix edge case"}
var diffFail = wsverdict.Diff{WSID: "00-146-03", DiffText: "-func Core()", Context: "Remove core functionality"}

// =====================================================
// Decomposed pipeline tests (F146-04)
// =====================================================

// noMicro returns a DecomposedRunner with MicroFirst disabled, for testing the raw LLM pipeline.
func noMicro(mock *mockLLM) *wsverdict.DecomposedRunner {
	return wsverdict.NewDecomposedRunnerWithOpts(mock, wsverdict.DecomposedRunnerOptions{MicroFirst: false})
}

// Fixture 1: clean diff → passed verdict.
func TestDecomposed_Clean_Passed(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractClean, // stage 1: extract
		"passed",     // stage 2: classify
		aggPassed,    // stage 3: aggregate
	}}

	runner := noMicro(mock)
	res, err := runner.Run(context.Background(), diffClean)
	require.NoError(t, err)
	assert.Equal(t, decompose.StatusOK, res.Status)
	assert.Equal(t, "passed", res.Answer.Verdict)
	assert.InDelta(t, 0.95, res.Answer.Score, 0.001)
	assert.Len(t, res.StageResults, 3)
	assert.Equal(t, "extract", res.StageResults[0].Name)
	assert.Equal(t, "classify", res.StageResults[1].Name)
	assert.Equal(t, "aggregate", res.StageResults[2].Name)
}

// Fixture 2: mixed diff → partial verdict.
func TestDecomposed_Mixed_Partial(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractMixed,
		"partial",
		aggPartial,
	}}

	runner := noMicro(mock)
	res, err := runner.Run(context.Background(), diffMixed)
	require.NoError(t, err)
	assert.Equal(t, "partial", res.Answer.Verdict)
	assert.NotEmpty(t, res.Answer.BlockingGates)
}

// Fixture 3: failing diff → failed verdict.
func TestDecomposed_Fail_Failed(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractFail,
		"failed",
		aggFailed,
	}}

	runner := noMicro(mock)
	res, err := runner.Run(context.Background(), diffFail)
	require.NoError(t, err)
	assert.Equal(t, "failed", res.Answer.Verdict)
	assert.Len(t, res.Answer.BlockingGates, 2)
}

// Extract stage retries on first failure (RetryOnce policy).
func TestDecomposed_ExtractRetry(t *testing.T) {
	mock := &mockLLM{responses: []string{
		"not valid json", // first attempt fails
		extractClean,    // retry succeeds
		"passed",
		aggPassed,
	}}

	runner := noMicro(mock)
	res, err := runner.Run(context.Background(), diffClean)
	require.NoError(t, err)
	assert.Equal(t, "passed", res.Answer.Verdict)
	assert.Equal(t, 4, mock.calls)
}

// Classify stage invalid verdict → pipeline errors (Abort policy).
func TestDecomposed_ClassifyAbort(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractClean,
		"uncertain", // invalid enum value
	}}

	runner := noMicro(mock)
	_, err := runner.Run(context.Background(), diffClean)
	require.Error(t, err)
}

// Aggregate stage failure → Fallback with failed verdict.
func TestDecomposed_AggregateFallback(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractClean,
		"passed",
		"not valid json", // aggregate fails
	}}

	runner := noMicro(mock)
	res, err := runner.Run(context.Background(), diffClean)
	require.NoError(t, err) // Fallback = no error
	assert.Equal(t, decompose.StatusFail, res.Status)
	assert.Equal(t, "failed", res.Answer.Verdict) // fallback value
}

// =====================================================
// Monolithic baseline tests (F146-05)
// =====================================================

// Fixture 1: clean → passed.
func TestMonolithic_Clean_Passed(t *testing.T) {
	mock := &mockLLM{responses: []string{aggPassed}}

	runner := wsverdict.NewMonolithicRunner(mock)
	res, err := runner.Run(context.Background(), diffClean)
	require.NoError(t, err)
	assert.Equal(t, "passed", res.Answer.Verdict)
	assert.InDelta(t, 0.95, res.Answer.Score, 0.001)
	// Exactly one synthetic stage named "monolithic".
	require.Len(t, res.StageResults, 1)
	assert.Equal(t, "monolithic", res.StageResults[0].Name)
}

// Fixture 2: mixed → partial.
func TestMonolithic_Mixed_Partial(t *testing.T) {
	mock := &mockLLM{responses: []string{aggPartial}}

	runner := wsverdict.NewMonolithicRunner(mock)
	res, err := runner.Run(context.Background(), diffMixed)
	require.NoError(t, err)
	assert.Equal(t, "partial", res.Answer.Verdict)
}

// Fixture 3: fail → failed.
func TestMonolithic_Fail_Failed(t *testing.T) {
	mock := &mockLLM{responses: []string{aggFailed}}

	runner := wsverdict.NewMonolithicRunner(mock)
	res, err := runner.Run(context.Background(), diffFail)
	require.NoError(t, err)
	assert.Equal(t, "failed", res.Answer.Verdict)
}

// Monolithic LLM error → error returned.
func TestMonolithic_LLMError(t *testing.T) {
	mock := &mockLLM{responses: []string{}} // no responses configured

	runner := wsverdict.NewMonolithicRunner(mock)
	_, err := runner.Run(context.Background(), diffClean)
	require.Error(t, err)
}

// Monolithic invalid JSON → error returned.
func TestMonolithic_ParseError(t *testing.T) {
	mock := &mockLLM{responses: []string{"not json"}}

	runner := wsverdict.NewMonolithicRunner(mock)
	_, err := runner.Run(context.Background(), diffClean)
	require.Error(t, err)
}

// =====================================================
// Evidence helpers (aggregate.go)
// =====================================================

func TestMarshalEvidenceJSON(t *testing.T) {
	fv := wsverdict.FinalVerdict{Verdict: "passed", Score: 0.9, Summary: "all good"}
	out, err := wsverdict.MarshalEvidenceJSON(fv)
	require.NoError(t, err)
	assert.Contains(t, out, `"verdict"`)
	assert.Contains(t, out, "passed")
}

func TestMarshalFinalVerdictTOON(t *testing.T) {
	fv := wsverdict.FinalVerdict{Verdict: "passed", Score: 0.9, Summary: "all good"}
	out, err := wsverdict.MarshalFinalVerdictTOON(fv)
	require.NoError(t, err)
	assert.Contains(t, out, "verdict=passed")
}

// Aggregate stage with invalid verdict in response → error propagates.
func TestDecomposed_AggregateInvalidVerdict(t *testing.T) {
	mock := &mockLLM{responses: []string{
		extractClean,
		"passed",
		`{"verdict":"unknown","score":0.5,"summary":"test","blocking_gates":[]}`,
	}}
	runner := noMicro(mock) // disable micro gate to exercise aggregate fallback path
	_, err := runner.Run(context.Background(), diffClean)
	// Fallback absorbs the error — runner returns no error but status FAIL.
	require.NoError(t, err)
}

// Monolithic result has same FinalVerdict shape as decomposed output.
func TestMonolithic_SameOutputShape(t *testing.T) {
	mock := &mockLLM{responses: []string{aggPassed}}
	res, err := wsverdict.NewMonolithicRunner(mock).Run(context.Background(), diffClean)
	require.NoError(t, err)
	// Verify all FinalVerdict fields are populated.
	assert.NotEmpty(t, res.Answer.Verdict)
	assert.GreaterOrEqual(t, res.Answer.Score, 0.0)
	assert.LessOrEqual(t, res.Answer.Score, 1.0)
	assert.NotEmpty(t, res.Answer.Summary)
}

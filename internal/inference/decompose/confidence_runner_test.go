package decompose_test

import (
	"context"
	"testing"

	"sdp_dev/internal/inference/confidence"
	"sdp_dev/internal/inference/confidence/constraint"
	"sdp_dev/internal/inference/decompose"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewConfidenceRunner_TypedWrapper exercises NewConfidenceRunner with a
// real confidence.Checker to cover the typed wrapper code path.
func TestNewConfidenceRunner_TypedWrapper(t *testing.T) {
	// Constraint with no schema and no invariants → always passes.
	strat, err := constraint.New[string](constraint.Options[string]{})
	require.NoError(t, err)

	checker, err := confidence.NewChecker[string](nil, []confidence.Strategy[string]{strat}, confidence.Policy{
		Weights:        map[string]float64{"constraint": 1.0},
		OKThreshold:    0.5,
		FailThreshold:  0.0,
		UnsureBehavior: confidence.UnsureRetryOnce,
	})
	require.NoError(t, err)

	runner := decompose.NewConfidenceRunner(checker)

	res, err := runner.Run(context.Background(), "input", "raw", "my-answer")
	require.NoError(t, err)
	assert.Equal(t, decompose.StatusOK, res.Status)
	assert.InDelta(t, 1.0, res.Score, 0.001)
}

// TestNewConfidenceRunner_WrongType ensures a type mismatch returns an error.
func TestNewConfidenceRunner_WrongType(t *testing.T) {
	strat, err := constraint.New[string](constraint.Options[string]{})
	require.NoError(t, err)
	checker, err := confidence.NewChecker[string](nil, []confidence.Strategy[string]{strat}, confidence.Policy{
		Weights:        map[string]float64{"constraint": 1.0},
		OKThreshold:    0.5,
		FailThreshold:  0.0,
		UnsureBehavior: confidence.UnsureRetryOnce,
	})
	require.NoError(t, err)
	runner := decompose.NewConfidenceRunner(checker)

	// Pass int instead of string → type mismatch error.
	_, err = runner.Run(context.Background(), "", "", 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected")
}

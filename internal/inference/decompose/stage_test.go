package decompose_test

import (
	"context"
	"errors"
	"testing"

	"sdp_dev/internal/inference/decompose"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStage_Name(t *testing.T) {
	s := decompose.NewStage[int, string]("convert", func(_ context.Context, n int) (string, decompose.StageTrace, error) {
		return "ok", decompose.StageTrace{}, nil
	})
	assert.Equal(t, "convert", s.Name())
}

func TestNewStage_Run_OK(t *testing.T) {
	s := decompose.NewStage[int, int]("double", func(_ context.Context, n int) (int, decompose.StageTrace, error) {
		return n * 2, decompose.StageTrace{TokensIn: 5}, nil
	})
	out, trace, err := s.Run(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, 14, out)
	assert.Equal(t, 5, trace.TokensIn)
}

func TestNewStage_Run_Error(t *testing.T) {
	boom := errors.New("stage failed")
	s := decompose.NewStage[int, int]("fail", func(_ context.Context, _ int) (int, decompose.StageTrace, error) {
		return 0, decompose.StageTrace{}, boom
	})
	_, _, err := s.Run(context.Background(), 1)
	require.ErrorIs(t, err, boom)
}

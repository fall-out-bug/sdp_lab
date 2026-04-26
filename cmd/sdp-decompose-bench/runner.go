package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sdp_dev/internal/inference/decompose"
	"sdp_dev/internal/inference/decompose/adapters/wsverdict"
	"sdp_dev/internal/inference/replayutil"
)

// benchResult holds the A/B comparison for one fixture.
type benchResult struct {
	Fixture    replayutil.Fixture
	MonoResult decompose.Result[wsverdict.FinalVerdict]
	MonoErr    error
	MonoMs     int64
	DecompResult decompose.Result[wsverdict.FinalVerdict]
	DecompErr    error
	DecompMs     int64
}

// runFixture runs both monolithic and decomposed pipelines on one fixture.
func runFixture(ctx context.Context, f replayutil.Fixture, monoRunner *wsverdict.MonolithicRunner, decompRunner *wsverdict.DecomposedRunner) benchResult {
	diff := fixtureToDiff(f)
	r := benchResult{Fixture: f}

	t0 := time.Now()
	r.MonoResult, r.MonoErr = monoRunner.Run(ctx, diff)
	r.MonoMs = time.Since(t0).Milliseconds()

	t1 := time.Now()
	r.DecompResult, r.DecompErr = decompRunner.Run(ctx, diff)
	r.DecompMs = time.Since(t1).Milliseconds()

	return r
}

// fixtureToDiff converts a corpus fixture to a wsverdict.Diff input.
// The raw JSON is used as the "diff text" (proxy for real diff content).
func fixtureToDiff(f replayutil.Fixture) wsverdict.Diff {
	wsID, _ := f.Data["ws_id"].(string)
	summary, _ := f.Data["existing_work_summary"].(string)
	return wsverdict.Diff{
		WSID:     wsID,
		DiffText: string(f.Raw),
		Context:  summary,
	}
}

// toRecord converts a benchResult to a RunRecord for evidence writing.
func toRecord(r benchResult) replayutil.RunRecord {
	monoStatus := verdictStatus(r.MonoResult, r.MonoErr)
	decompStatus := verdictStatus(r.DecompResult, r.DecompErr)

	monoTokens := totalTokens(r.MonoResult)
	decompTokens := totalTokens(r.DecompResult)

	monoCost := totalCost(r.MonoResult)
	decompCost := totalCost(r.DecompResult)

	return replayutil.RunRecord{
		ID:               r.Fixture.ID,
		Category:         r.Fixture.Category,
		MonolithicStatus: monoStatus,
		DecomposedStatus: decompStatus,
		LatencyMonoMs:    r.MonoMs,
		LatencyDecompMs:  r.DecompMs,
		TokensMono:       monoTokens,
		TokensDecomp:     decompTokens,
		CostMonoUSD:      monoCost,
		CostDecompUSD:    decompCost,
	}
}

func verdictStatus(res decompose.Result[wsverdict.FinalVerdict], err error) string {
	if err != nil {
		return "error"
	}
	if res.Answer.Verdict != "" {
		return res.Answer.Verdict
	}
	return string(res.Status)
}

func totalTokens(res decompose.Result[wsverdict.FinalVerdict]) int {
	return res.Trace.TokensIn + res.Trace.TokensOut
}

func totalCost(res decompose.Result[wsverdict.FinalVerdict]) float64 {
	return res.Trace.CostUSD
}

// accuracyMatch returns true if the pipeline result verdict matches the golden.
func accuracyMatch(result decompose.Result[wsverdict.FinalVerdict], err error, golden string) bool {
	if err != nil {
		return false
	}
	return result.Answer.Verdict == golden
}

// toonSaving returns the ratio of TOON-marshalled FinalVerdict to JSON-marshalled.
func toonSaving(fv wsverdict.FinalVerdict) (float64, error) {
	jsonBytes, err := json.MarshalIndent(fv, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("json marshal: %w", err)
	}
	toonOut, err := wsverdict.MarshalFinalVerdictTOON(fv)
	if err != nil {
		return 0, fmt.Errorf("toon marshal: %w", err)
	}
	if len(jsonBytes) == 0 {
		return 1.0, nil
	}
	return float64(len(toonOut)) / float64(len(jsonBytes)), nil
}

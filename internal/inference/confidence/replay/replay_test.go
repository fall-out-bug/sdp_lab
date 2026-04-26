package replay_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/inference/confidence"
	"sdp_dev/internal/inference/confidence/adapters/wsverdict"
	"sdp_dev/internal/inference/confidence/replay"
)

func loadSchema(t *testing.T) []byte {
	t.Helper()
	wd, _ := os.Getwd()
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		dir = filepath.Dir(dir)
	}
	data, err := os.ReadFile(filepath.Join(dir, "schema", "ws-verdict.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	return data
}

func corpusRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		dir = filepath.Dir(dir)
	}
	return filepath.Join(dir, "internal", "inference", "confidence", "testdata")
}

func TestRunWsVerdictCorpus(t *testing.T) {
	checker, err := wsverdict.New(wsverdict.Options{SchemaJSON: loadSchema(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r := &replay.Runner[wsverdict.Verdict]{
		Checker:   checker,
		CorpusDir: filepath.Join(corpusRoot(t), "ws-verdict"),
		Verify: func(ctx context.Context, raw []byte) (confidence.Result[wsverdict.Verdict], error) {
			return wsverdict.Verify(ctx, checker, string(raw), raw)
		},
	}
	rep, err := r.Run(context.Background(), "ws-verdict")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.CallSite != "ws-verdict" {
		t.Errorf("CallSite = %q", rep.CallSite)
	}
	if len(rep.Categories) != 3 {
		t.Errorf("Categories len = %d, want 3", len(rep.Categories))
	}

	// Find the adversarial category and verify rejection rate ≥ 0.8.
	var adv, corr replay.CategoryMetrics
	for _, c := range rep.Categories {
		switch c.Category {
		case replay.Adversarial:
			adv = c
		case replay.Correct:
			corr = c
		}
	}
	if adv.N == 0 {
		t.Fatal("adversarial corpus empty")
	}
	if adv.RejectionRate < 0.8 {
		t.Errorf("adversarial rejection %.2f < 0.80 — F144-08 acceptance gate", adv.RejectionRate)
	}
	if corr.RejectionRate > 0.02 {
		t.Errorf("correct false-FAIL %.2f > 0.02 — F144-08 acceptance gate", corr.RejectionRate)
	}
}

func TestRenderMarkdown(t *testing.T) {
	reports := []replay.CallSiteReport{
		{
			CallSite: "test",
			Categories: []replay.CategoryMetrics{
				{Category: replay.Correct, N: 10, OK: 10, RejectionRate: 0.0},
				{Category: replay.Adversarial, N: 5, Fail: 5, RejectionRate: 1.0},
			},
		},
	}
	md := replay.RenderMarkdown(reports)
	if !strings.Contains(md, "# F144 Confidence Replay Report") {
		t.Errorf("markdown missing title")
	}
	if !strings.Contains(md, "PASS") {
		t.Errorf("markdown missing verdict line on clean case")
	}
}

func TestRenderMarkdownFail(t *testing.T) {
	reports := []replay.CallSiteReport{
		{
			CallSite: "test",
			Categories: []replay.CategoryMetrics{
				{Category: replay.Correct, N: 10, Fail: 5, RejectionRate: 0.5}, // bad
				{Category: replay.Adversarial, N: 5, Fail: 1, RejectionRate: 0.2},
			},
		},
	}
	md := replay.RenderMarkdown(reports)
	if !strings.Contains(md, "FAIL") {
		t.Errorf("markdown should report FAIL verdict")
	}
}

func TestRenderMarkdownFailsOnFixtureErrors(t *testing.T) {
	reports := []replay.CallSiteReport{
		{
			CallSite: "test",
			Categories: []replay.CategoryMetrics{
				{Category: replay.Correct, N: 10, OK: 9, Errors: 1, RejectionRate: 0.0},
				{Category: replay.Adversarial, N: 5, Fail: 5, RejectionRate: 1.0},
			},
		},
	}
	md := replay.RenderMarkdown(reports)
	if !strings.Contains(md, "FAIL") || !strings.Contains(md, "fixture errors") {
		t.Errorf("markdown should fail on fixture errors, got:\n%s", md)
	}
}

func TestPercentileEdges(t *testing.T) {
	// Indirect — exposed via Run with one fixture per category.
	// The single-element latency slice should produce p50=p95=that value.
	checker, _ := wsverdict.New(wsverdict.Options{SchemaJSON: loadSchema(t)})
	r := &replay.Runner[wsverdict.Verdict]{
		Checker:   checker,
		CorpusDir: filepath.Join(corpusRoot(t), "ws-verdict"),
		Verify: func(ctx context.Context, raw []byte) (confidence.Result[wsverdict.Verdict], error) {
			return wsverdict.Verify(ctx, checker, string(raw), raw)
		},
	}
	rep, err := r.Run(context.Background(), "ws-verdict")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range rep.Categories {
		if c.N == 0 {
			continue
		}
		if c.LatencyMsP95 < c.LatencyMsP50 {
			t.Errorf("category %s: p95 %d < p50 %d", c.Category, c.LatencyMsP95, c.LatencyMsP50)
		}
	}
}

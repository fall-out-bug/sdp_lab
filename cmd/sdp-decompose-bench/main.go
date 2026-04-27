// sdp-decompose-bench runs an A/B benchmark comparing the monolithic ws-verdict
// pipeline against the 3-stage decomposed pipeline (F146) over the F144 corpus.
//
// Usage:
//
//	sdp-decompose-bench --corpus <path> --out <report.md>
//	sdp-decompose-bench --corpus internal/inference/confidence/testdata/ws-verdict \
//	                    --out docs/research/2026-04-26-f146-decomposition-replay-report.md
//
// The bench requires OPENROUTER_API_KEY to be set for real LLM calls.
// In dry-run mode (--dry-run), a deterministic mock LLM is used and results
// are annotated as simulated.
//
// Exit codes:
//
//	0  bench complete (informational thresholds may or may not pass)
//	2  internal error (corpus missing, output directory unwritable)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose/adapters/wsverdict"
	"github.com/fall-out-bug/sdp_lab/internal/inference/replayutil"
)

func main() {
	corpusPath := flag.String("corpus", "internal/inference/confidence/testdata/ws-verdict", "Path to ws-verdict corpus directory")
	outPath := flag.String("out", "docs/research/2026-04-26-f146-decomposition-replay-report.md", "Output markdown report path")
	dryRun := flag.Bool("dry-run", false, "Use deterministic mock LLM instead of real API")
	evidenceDir := flag.String("evidence-dir", "internal/build/.sdp/evidence", "Directory for per-run JSON/CSV evidence")
	flag.Parse()

	ctx := context.Background()

	fixtures, err := replayutil.LoadCorpus(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bench: load corpus: %v\n", err)
		os.Exit(2)
	}
	if len(fixtures) == 0 {
		fmt.Fprintf(os.Stderr, "bench: no fixtures found in %s\n", *corpusPath)
		os.Exit(2)
	}
	fmt.Printf("bench: loaded %d fixtures from %s\n", len(fixtures), *corpusPath)

	var client wsverdict.LLMCaller
	if *dryRun || os.Getenv("OPENROUTER_API_KEY") == "" {
		fmt.Println("bench: using dry-run mock LLM (no API key or --dry-run)")
		client = newDryRunClient(fixtures)
	} else {
		client = newOpenRouterClient(os.Getenv("OPENROUTER_API_KEY"))
	}

	monoRunner := wsverdict.NewMonolithicRunner(client)
	decompRunner := wsverdict.NewDecomposedRunner(client)

	// fixtureSettable is satisfied by dryRunClient to bind golden verdict per fixture.
	type fixtureSettable interface {
		SetFixture(replayutil.Fixture)
	}

	var results []benchResult
	var records []replayutil.RunRecord
	for _, f := range fixtures {
		if fs, ok := client.(fixtureSettable); ok {
			fs.SetFixture(f)
		}
		fmt.Printf("  running fixture %s ...\n", f.ID)
		r := runFixture(ctx, f, monoRunner, decompRunner)
		results = append(results, r)
		records = append(records, toRecord(r))
	}

	runID := replayutil.NewRunID()
	evidence := replayutil.AggregateEvidence{
		RunID:   runID,
		Records: records,
	}
	if err := replayutil.WriteEvidence(*evidenceDir, runID, evidence); err != nil {
		fmt.Fprintf(os.Stderr, "bench: write evidence: %v\n", err)
		// Non-fatal: continue to report.
	}

	isDryRun := *dryRun || os.Getenv("OPENROUTER_API_KEY") == ""
	report := renderReport(reportData{
		RunID:    runID,
		Corpus:   *corpusPath,
		DryRun:   isDryRun,
		Records:  records,
		Results:  results,
		Fixtures: fixtures,
	})

	outDir := filepath.Dir(*outPath)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "bench: create output dir: %v\n", err)
		os.Exit(2)
	}
	if err := os.WriteFile(*outPath, []byte(report), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "bench: write report: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("bench: report written to %s\n", *outPath)
}

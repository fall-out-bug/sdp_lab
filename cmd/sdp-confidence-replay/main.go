//go:build sdp_experimental

// sdp-confidence-replay drives the F144 confidence Checker against the
// fixture corpus under internal/inference/confidence/testdata/ and emits a
// markdown report plus per-fixture JSON evidence.
//
// Usage:
//
//	sdp-confidence-replay [-out path]   default: docs/research/2026-04-26-f144-confidence-replay-report.md
//	sdp-confidence-replay -json         dump JSON instead of markdown
//
// Exit codes:
//
//	0  acceptance gates pass on every call-site
//	1  one or more call-sites fail acceptance (adversarial rej < 0.80 or correct false-FAIL > 0.02)
//	2  internal error (corpus missing, schema unreadable)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence"
	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence/adapters/wsverdict"
	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence/replay"
)

func main() {
	out := flag.String("out", "docs/research/2026-04-26-f144-confidence-replay-report.md", "output markdown path")
	jsonOut := flag.Bool("json", false, "dump JSON to stdout instead of writing markdown")
	corpusRoot := flag.String("corpus", "internal/inference/confidence/testdata", "corpus root")
	schemaPath := flag.String("schema", "schema/ws-verdict.schema.json", "ws-verdict schema path")
	flag.Parse()

	ctx := context.Background()
	reports, err := runAll(ctx, *corpusRoot, *schemaPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	if *jsonOut {
		for _, r := range reports {
			b, err := replay.MarshalJSON(r)
			if err != nil {
				fmt.Fprintln(os.Stderr, "marshal:", err)
				os.Exit(2)
			}
			fmt.Println(string(b))
		}
	} else {
		md := replay.RenderMarkdown(reports)
		if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "mkdir:", err)
			os.Exit(2)
		}
		if err := os.WriteFile(*out, []byte(md), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(2)
		}
		fmt.Println("wrote", *out)
	}

	if !allPass(reports) {
		os.Exit(1)
	}
}

// runAll currently runs the ws-verdict call-site (adapter best-coverage
// among Wave 2). Architect/dispatch corpora are intentionally minimal in
// F144-08; extending them is future work tracked separately.
func runAll(ctx context.Context, corpusRoot, schemaPath string) ([]replay.CallSiteReport, error) {
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}
	checker, err := wsverdict.New(wsverdict.Options{SchemaJSON: schema})
	if err != nil {
		return nil, fmt.Errorf("new wsverdict checker: %w", err)
	}
	r := &replay.Runner[wsverdict.Verdict]{
		Checker:   checker,
		CorpusDir: filepath.Join(corpusRoot, "ws-verdict"),
		Verify: func(ctx context.Context, raw []byte) (confidence.Result[wsverdict.Verdict], error) {
			return wsverdict.Verify(ctx, checker, string(raw), raw)
		},
	}
	wsv, err := r.Run(ctx, "ws-verdict")
	if err != nil {
		return nil, fmt.Errorf("run ws-verdict: %w", err)
	}
	return []replay.CallSiteReport{wsv}, nil
}

func allPass(reports []replay.CallSiteReport) bool {
	for _, r := range reports {
		var advRej, corrFalseFail float64
		var errors int
		for _, c := range r.Categories {
			errors += c.Errors
			switch c.Category {
			case replay.Adversarial:
				advRej = c.RejectionRate
			case replay.Correct:
				corrFalseFail = c.RejectionRate
			}
		}
		if errors > 0 || advRej < 0.80 || corrFalseFail > 0.02 {
			return false
		}
	}
	return true
}

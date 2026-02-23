package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAgentOutput(t *testing.T) {
	// Simple format: role + content
	data := []byte(`{"role":"user","content":"hello"}
{"role":"assistant","content":"agent says hi"}`)
	out := extractAgentOutput(data)
	if out != "agent says hi\n" {
		t.Errorf("got %q", out)
	}
}

func TestRunCase_KnownBad(t *testing.T) {
	tmp := t.TempDir()
	// Transcript with forbidden patterns
	os.WriteFile(filepath.Join(tmp, "bad.jsonl"), []byte(`{"role":"assistant","content":"Next steps: 1. approve and merge"}`), 0o644)
	c := &Case{
		Name:             "bad",
		InputTranscript:  "bad.jsonl",
		ForbiddenPatterns: []string{"Next steps", "approve and merge"},
		RequiredPatterns: []string{},
	}
	r := RunCase(c, tmp)
	if r.Pass {
		t.Error("expected FAIL for known-bad transcript")
	}
}

func TestRunCase_KnownGood(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "good.jsonl"), []byte(`{"role":"assistant","content":"CI GREEN - @oneshot complete"}`), 0o644)
	c := &Case{
		Name:              "good",
		InputTranscript:   "good.jsonl",
		ForbiddenPatterns: []string{"Next steps"},
		RequiredPatterns:  []string{"CI GREEN"},
	}
	r := RunCase(c, tmp)
	if !r.Pass {
		t.Errorf("expected PASS for known-good transcript: %s", r.Reason)
	}
}

func TestRun_OneshotEvals(t *testing.T) {
	// Run from project root so testdata paths resolve
	root, _ := os.Getwd()
	for _, d := range []string{"internal/eval", "eval"} {
		if _, err := os.Stat(filepath.Join(root, d)); err == nil {
			root = filepath.Dir(root)
			break
		}
	}
	// Find project root (has testdata/eval)
	for {
		if _, err := os.Stat(filepath.Join(root, "testdata", "eval")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Skip("project root not found")
		}
		root = parent
	}
	casesDir := filepath.Join(root, "internal", "eval", "cases")
	results, err := Run(root, casesDir, "oneshot")
	if err != nil {
		t.Fatal(err)
	}
	passed := 0
	for _, r := range results {
		if r.Pass {
			passed++
		}
	}
	// We expect: 3 FAIL (known-bad) + 2 PASS (known-good) = 5 total
	if len(results) != 5 {
		t.Errorf("expected 5 cases, got %d", len(results))
	}
	if passed != 2 {
		t.Errorf("expected 2 pass (ci-green, uses-ci-loop), got %d", passed)
	}
}

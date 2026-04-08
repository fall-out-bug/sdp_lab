# Phase 4a VALIDATE (Desk Research) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** For each of the top 3 RAT-ranked assumptions, run adversarial desk-research prompting (evidence FOR + AGAINST) and compute a ClaimVerdict per assumption, then synthesise a final GO / PIVOT / KILL verdict.

**Architecture:** Three components land in sequence — (1) `validate.go` owns all types and the two LLM calls (per-claim adversarial + synthesis), (2) `artifacts.go` gains `Session.Validation` and `writeValidation()`, (3) `cmd_discover.go` calls Phase 4a after Checkpoint C and prints the final verdict prominently. All calls use `DefaultReasonerModel` (openai/gpt-5.4-mini). `NeedsExperiment` is set true if any claim returns `insufficient_data`, which will later trigger Phase 4b.

**Tech Stack:** Go 1.23, `internal/discovery` package, OpenRouter API via `LLMClient`, `DefaultReasonerModel = "openai/gpt-5.4-mini"`, `encoding/json`, `strings`, `fmt`

---

### Task 1: Types + pure-logic unit test

**Files:**
- Create: `internal/discovery/validate.go`
- Create: `internal/discovery/validate_test.go`

**Step 1: Write the failing test**

```go
// internal/discovery/validate_test.go
package discovery_test

import (
	"testing"

	"sdp_dev/internal/discovery"
)

func TestNeedsExperiment_FalseWhenAllSupported(t *testing.T) {
	claims := []discovery.ClaimValidation{
		{Verdict: discovery.VerdictSupported},
		{Verdict: discovery.VerdictContradicted},
	}
	if discovery.NeedsExperimentFromClaims(claims) {
		t.Error("expected false: no insufficient_data verdict")
	}
}

func TestNeedsExperiment_TrueWhenAnyInsufficientData(t *testing.T) {
	claims := []discovery.ClaimValidation{
		{Verdict: discovery.VerdictSupported},
		{Verdict: discovery.VerdictInsufficientData},
	}
	if !discovery.NeedsExperimentFromClaims(claims) {
		t.Error("expected true: has insufficient_data verdict")
	}
}

func TestRenderClaimsForSynthesis_ContainsRankAndVerdict(t *testing.T) {
	claims := []discovery.ClaimValidation{
		{
			Claim:      "founders need validated ideas before coding",
			RATRank:    1,
			Verdict:    discovery.VerdictSupported,
			Confidence: 0.8,
			Notes:      "ample survey data",
		},
	}
	out := discovery.RenderClaimsForSynthesis(claims)
	if out == "" {
		t.Fatal("empty render output")
	}
	for _, want := range []string{"Rank 1", "SUPPORTED", "founders need validated ideas"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}
```

Wait — need `strings` import. Add it:

```go
import (
	"strings"
	"testing"

	"sdp_dev/internal/discovery"
)
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... -run "TestNeedsExperiment|TestRenderClaims" -v
```
Expected: FAIL — `discovery.VerdictSupported`, `ClaimValidation`, `NeedsExperimentFromClaims`, `RenderClaimsForSynthesis` undefined.

**Step 3: Write minimal implementation**

```go
// internal/discovery/validate.go
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ClaimVerdict is the per-assumption verdict from desk research.
type ClaimVerdict string

const (
	VerdictSupported       ClaimVerdict = "supported"
	VerdictContradicted    ClaimVerdict = "contradicted"
	VerdictInsufficientData ClaimVerdict = "insufficient_data"
)

// FinalVerdict is the overall product GO / PIVOT / KILL recommendation.
type FinalVerdict string

const (
	VerdictGO    FinalVerdict = "GO"
	VerdictPIVOT FinalVerdict = "PIVOT"
	VerdictKILL  FinalVerdict = "KILL"
)

// Evidence is one piece of evidence for or against an assumption.
type Evidence struct {
	Direction  string `json:"direction"`   // "for" | "against"
	Statement  string `json:"statement"`
	SourceURL  string `json:"source_url"`
	IsEstimate bool   `json:"is_estimate"`
}

// ClaimValidation is the desk-research result for one assumption.
type ClaimValidation struct {
	Claim      string       `json:"claim"`
	RATRank    int          `json:"rat_rank"`
	Evidence   []Evidence   `json:"evidence"`
	Verdict    ClaimVerdict `json:"verdict"`
	Confidence float64      `json:"confidence"`
	Notes      string       `json:"notes"`
}

// ValidationResult is the output of Phase 4a VALIDATE.
type ValidationResult struct {
	Claims          []ClaimValidation `json:"claims"`
	FinalVerdict    FinalVerdict      `json:"final_verdict"`
	VerdictReason   string            `json:"verdict_reason"`
	PivotSuggestion string            `json:"pivot_suggestion,omitempty"`
	KillReason      string            `json:"kill_reason,omitempty"`
	NeedsExperiment bool              `json:"needs_experiment"`
	CostUSD         float64           `json:"cost_usd"`
}

// NeedsExperimentFromClaims returns true if any claim has insufficient_data verdict.
func NeedsExperimentFromClaims(claims []ClaimValidation) bool {
	for _, c := range claims {
		if c.Verdict == VerdictInsufficientData {
			return true
		}
	}
	return false
}

// RenderClaimsForSynthesis formats claims as readable text for the synthesis prompt.
func RenderClaimsForSynthesis(claims []ClaimValidation) string {
	var b strings.Builder
	for _, c := range claims {
		fmt.Fprintf(&b, "Rank %d [RAT]: %q → %s (confidence: %.2f)\n",
			c.RATRank, c.Claim, strings.ToUpper(string(c.Verdict)), c.Confidence)
		if c.Notes != "" {
			fmt.Fprintf(&b, "  Notes: %s\n", c.Notes)
		}
		for _, e := range c.Evidence {
			fmt.Fprintf(&b, "  [%s] %s", strings.ToUpper(e.Direction), e.Statement)
			if e.IsEstimate {
				fmt.Fprintf(&b, " (estimate)")
			}
			fmt.Fprintf(&b, "\n")
		}
		fmt.Fprintf(&b, "\n")
	}
	return strings.TrimSpace(b.String())
}

// Validate is declared here so the file compiles; body added in Task 2.
func Validate(ctx context.Context, c *LLMClient, frame *FrameResult, h *HypothesisResult) (*ValidationResult, error) {
	_ = ctx
	_ = c
	_ = frame
	_ = h
	return nil, fmt.Errorf("not implemented")
}
```

Note: the `context`, `encoding/json` imports will be used in Task 2 — they're declared early to avoid having to add them later. If the linter complains about unused imports, add a blank import `_ = json.Marshal` sentinel or just add the imports in Task 2 instead. Keep the file compiling clean by only importing what's used.

Revised minimal version that compiles without unused imports:

```go
package discovery

import (
	"context"
	"fmt"
	"strings"
)

type ClaimVerdict string

const (
	VerdictSupported        ClaimVerdict = "supported"
	VerdictContradicted     ClaimVerdict = "contradicted"
	VerdictInsufficientData ClaimVerdict = "insufficient_data"
)

type FinalVerdict string

const (
	VerdictGO    FinalVerdict = "GO"
	VerdictPIVOT FinalVerdict = "PIVOT"
	VerdictKILL  FinalVerdict = "KILL"
)

type Evidence struct {
	Direction  string `json:"direction"`
	Statement  string `json:"statement"`
	SourceURL  string `json:"source_url"`
	IsEstimate bool   `json:"is_estimate"`
}

type ClaimValidation struct {
	Claim      string       `json:"claim"`
	RATRank    int          `json:"rat_rank"`
	Evidence   []Evidence   `json:"evidence"`
	Verdict    ClaimVerdict `json:"verdict"`
	Confidence float64      `json:"confidence"`
	Notes      string       `json:"notes"`
}

type ValidationResult struct {
	Claims          []ClaimValidation `json:"claims"`
	FinalVerdict    FinalVerdict      `json:"final_verdict"`
	VerdictReason   string            `json:"verdict_reason"`
	PivotSuggestion string            `json:"pivot_suggestion,omitempty"`
	KillReason      string            `json:"kill_reason,omitempty"`
	NeedsExperiment bool              `json:"needs_experiment"`
	CostUSD         float64           `json:"cost_usd"`
}

// NeedsExperimentFromClaims returns true if any claim has insufficient_data verdict.
func NeedsExperimentFromClaims(claims []ClaimValidation) bool {
	for _, c := range claims {
		if c.Verdict == VerdictInsufficientData {
			return true
		}
	}
	return false
}

// RenderClaimsForSynthesis formats claims as readable text for the synthesis prompt.
func RenderClaimsForSynthesis(claims []ClaimValidation) string {
	var b strings.Builder
	for _, c := range claims {
		fmt.Fprintf(&b, "Rank %d: %q → %s (confidence: %.2f)\n",
			c.RATRank, c.Claim, strings.ToUpper(string(c.Verdict)), c.Confidence)
		if c.Notes != "" {
			fmt.Fprintf(&b, "  Notes: %s\n", c.Notes)
		}
		for _, e := range c.Evidence {
			fmt.Fprintf(&b, "  [%s] %s", strings.ToUpper(e.Direction), e.Statement)
			if e.IsEstimate {
				fmt.Fprintf(&b, " (estimate)")
			}
			fmt.Fprintf(&b, "\n")
		}
		fmt.Fprintf(&b, "\n")
	}
	return strings.TrimSpace(b.String())
}

// Validate performs adversarial desk-research validation of the top RAT assumptions.
// Body is added in Task 2.
func Validate(ctx context.Context, c *LLMClient, frame *FrameResult, h *HypothesisResult) (*ValidationResult, error) {
	_, _, _ = ctx, c, frame
	_ = h
	return nil, fmt.Errorf("validate: not implemented")
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/discovery/... -run "TestNeedsExperiment|TestRenderClaims" -v
```
Expected: PASS (3 tests).

**Step 5: Build check**

```bash
go build ./...
```
Expected: no errors.

**Step 6: Commit**

```bash
git add internal/discovery/validate.go internal/discovery/validate_test.go
git commit -m "feat: add Phase 4a VALIDATE types, NeedsExperimentFromClaims, RenderClaimsForSynthesis"
```

---

### Task 2: Validate() — per-claim adversarial LLM + synthesis call

**Files:**
- Modify: `internal/discovery/validate.go` (replace stub Validate with real implementation)

**Step 1: Write the integration test first (will be skipped without API key)**

Add to `internal/discovery/validate_test.go`:

```go
func TestValidate_ProducesVerdict(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
	frame := &discovery.FrameResult{
		ProblemStatement: "developers spend hours on manual product discovery before writing specs",
		Jobs:             []string{"validate ideas cheaply before investing in implementation"},
		Appetite:         "medium",
		RawIdea:          "automate product discovery using AI agents",
	}
	h := &discovery.HypothesisResult{
		Assumptions: []discovery.Assumption{
			{Statement: "solo founders avoid expensive discovery cycles because they lack time and budget", RATRank: 1, RATScore: 9, RiskLevel: "high", Uncertainty: "high"},
			{Statement: "LLM-generated validation is trusted enough to influence go/no-go decisions", RATRank: 2, RATScore: 6, RiskLevel: "high", Uncertainty: "medium"},
			{Statement: "a CLI tool fits the workflow of indie developers better than a web UI", RATRank: 3, RATScore: 4, RiskLevel: "medium", Uncertainty: "medium"},
		},
	}
	result, err := discovery.Validate(context.Background(), c, frame, h)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(result.Claims) == 0 {
		t.Error("no claims validated")
	}
	if len(result.Claims) > 3 {
		t.Errorf("expected at most 3 claims, got %d", len(result.Claims))
	}
	if result.FinalVerdict == "" {
		t.Error("empty final verdict")
	}
	validVerdicts := map[discovery.FinalVerdict]bool{
		discovery.VerdictGO:    true,
		discovery.VerdictPIVOT: true,
		discovery.VerdictKILL:  true,
	}
	if !validVerdicts[result.FinalVerdict] {
		t.Errorf("invalid final verdict: %q", result.FinalVerdict)
	}
	if result.VerdictReason == "" {
		t.Error("empty verdict reason")
	}
	for _, cv := range result.Claims {
		if len(cv.Evidence) == 0 {
			t.Errorf("claim rank %d has no evidence", cv.RATRank)
		}
		validClaimVerdicts := map[discovery.ClaimVerdict]bool{
			discovery.VerdictSupported:        true,
			discovery.VerdictContradicted:     true,
			discovery.VerdictInsufficientData: true,
		}
		if !validClaimVerdicts[cv.Verdict] {
			t.Errorf("claim rank %d has invalid verdict %q", cv.RATRank, cv.Verdict)
		}
	}
	t.Logf("final verdict: %s — %s", result.FinalVerdict, result.VerdictReason)
	t.Logf("cost: $%.5f", result.CostUSD)
}
```

Also add imports at top of file:
```go
import (
	"context"
	"os"
	"strings"
	"testing"

	"sdp_dev/internal/discovery"
)
```

**Step 2: Run integration test — expect FAIL (not implemented)**

```bash
go test ./internal/discovery/... -run TestValidate_ProducesVerdict -v
```
Expected: FAIL — `validate: not implemented`.

**Step 3: Implement Validate() and helpers**

Replace the stub `Validate` in `internal/discovery/validate.go` with the full implementation. Also add the two prompt consts and the two helper functions. Here is the complete final file (replaces the stub file from Task 1):

```go
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ClaimVerdict is the per-assumption verdict from desk research.
type ClaimVerdict string

const (
	VerdictSupported        ClaimVerdict = "supported"
	VerdictContradicted     ClaimVerdict = "contradicted"
	VerdictInsufficientData ClaimVerdict = "insufficient_data"
)

// FinalVerdict is the overall product GO / PIVOT / KILL recommendation.
type FinalVerdict string

const (
	VerdictGO    FinalVerdict = "GO"
	VerdictPIVOT FinalVerdict = "PIVOT"
	VerdictKILL  FinalVerdict = "KILL"
)

// Evidence is one piece of evidence for or against an assumption.
type Evidence struct {
	Direction  string `json:"direction"`   // "for" | "against"
	Statement  string `json:"statement"`
	SourceURL  string `json:"source_url"`
	IsEstimate bool   `json:"is_estimate"`
}

// ClaimValidation is the desk-research result for one assumption.
type ClaimValidation struct {
	Claim      string       `json:"claim"`
	RATRank    int          `json:"rat_rank"`
	Evidence   []Evidence   `json:"evidence"`
	Verdict    ClaimVerdict `json:"verdict"`
	Confidence float64      `json:"confidence"`
	Notes      string       `json:"notes"`
}

// ValidationResult is the output of Phase 4a VALIDATE.
type ValidationResult struct {
	Claims          []ClaimValidation `json:"claims"`
	FinalVerdict    FinalVerdict      `json:"final_verdict"`
	VerdictReason   string            `json:"verdict_reason"`
	PivotSuggestion string            `json:"pivot_suggestion,omitempty"`
	KillReason      string            `json:"kill_reason,omitempty"`
	NeedsExperiment bool              `json:"needs_experiment"`
	CostUSD         float64           `json:"cost_usd"`
}

// NeedsExperimentFromClaims returns true if any claim has insufficient_data verdict.
func NeedsExperimentFromClaims(claims []ClaimValidation) bool {
	for _, c := range claims {
		if c.Verdict == VerdictInsufficientData {
			return true
		}
	}
	return false
}

// RenderClaimsForSynthesis formats claims as readable text for the synthesis prompt.
func RenderClaimsForSynthesis(claims []ClaimValidation) string {
	var b strings.Builder
	for _, c := range claims {
		fmt.Fprintf(&b, "Rank %d: %q → %s (confidence: %.2f)\n",
			c.RATRank, c.Claim, strings.ToUpper(string(c.Verdict)), c.Confidence)
		if c.Notes != "" {
			fmt.Fprintf(&b, "  Notes: %s\n", c.Notes)
		}
		for _, e := range c.Evidence {
			fmt.Fprintf(&b, "  [%s] %s", strings.ToUpper(e.Direction), e.Statement)
			if e.IsEstimate {
				fmt.Fprintf(&b, " (estimate)")
			}
			fmt.Fprintf(&b, "\n")
		}
		fmt.Fprintf(&b, "\n")
	}
	return strings.TrimSpace(b.String())
}

const validateClaimSystemPrompt = `You are an adversarial product analyst. Find evidence BOTH FOR and AGAINST the given assumption. Be intellectually honest — do not favour confirmation. Cite real studies or data where you can; flag all estimates clearly.
Respond ONLY with valid JSON — no markdown, no explanation.`

const validateClaimUserPromptTpl = `Evaluate this product assumption using desk research.

ASSUMPTION (RAT rank %d, score %.0f): %s
PROBLEM CONTEXT: %s

Find 3–4 pieces of evidence FOR this assumption and 3–4 pieces AGAINST it.
For each piece of evidence:
- direction: "for" or "against"
- statement: one specific, concrete finding (not vague)
- source_url: URL if you know a real one; empty string if not — do NOT fabricate URLs
- is_estimate: true if pattern-based reasoning; false only if citing a known study/report/dataset

After listing evidence, assess the verdict:
- "supported": preponderance of credible evidence supports the assumption
- "contradicted": preponderance of credible evidence refutes it
- "insufficient_data": evidence is mixed, weak, or mostly estimated

Return JSON:
{"evidence":[{"direction":"for|against","statement":"...","source_url":"","is_estimate":true}],"verdict":"supported|contradicted|insufficient_data","confidence":0.7,"notes":"one-sentence synthesis"}`

const validateSynthesisSystemPrompt = `You are a product strategist giving a final GO / PIVOT / KILL recommendation.
Respond ONLY with valid JSON — no markdown, no explanation.`

const validateSynthesisUserPromptTpl = `Based on validated product assumptions, give a final verdict.

PROBLEM: %s

VALIDATED CLAIMS:
%s

Rules:
- GO: majority of high-RAT claims are "supported" with reasonable confidence (>= 0.6)
- KILL: the rank-1 claim is "contradicted" with high confidence (>= 0.7), OR majority are contradicted
- PIVOT: mixed — some supported, some contradicted or insufficient_data; a pivot direction is visible

Return JSON:
{"final_verdict":"GO|PIVOT|KILL","verdict_reason":"2-3 sentences referencing specific claims","pivot_suggestion":"what to pivot to (only if PIVOT, else omit)","kill_reason":"why kill (only if KILL, else omit)"}`

const validateTopN = 3

// validateClaim runs one adversarial desk-research call for a single assumption.
// Returns the ClaimValidation and the LLM cost in USD.
func validateClaim(ctx context.Context, c *LLMClient, problem string, a Assumption) (ClaimValidation, float64, error) {
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultReasonerModel,
		Messages: []Message{
			{Role: "system", Content: validateClaimSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(validateClaimUserPromptTpl,
				a.RATRank, a.RATScore, a.Statement, problem)},
		},
		MaxTokens:   1500,
		Temperature: 0.1,
	})
	if err != nil {
		return ClaimValidation{}, 0, fmt.Errorf("validate claim llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var raw struct {
		Evidence   []Evidence   `json:"evidence"`
		Verdict    ClaimVerdict `json:"verdict"`
		Confidence float64      `json:"confidence"`
		Notes      string       `json:"notes"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return ClaimValidation{}, 0, fmt.Errorf("validate claim parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}

	return ClaimValidation{
		Claim:      a.Statement,
		RATRank:    a.RATRank,
		Evidence:   raw.Evidence,
		Verdict:    raw.Verdict,
		Confidence: raw.Confidence,
		Notes:      raw.Notes,
	}, resp.CostUSD, nil
}

// synthResult is a helper for unmarshalling the synthesis LLM response.
type synthResult struct {
	FinalVerdict    FinalVerdict `json:"final_verdict"`
	VerdictReason   string       `json:"verdict_reason"`
	PivotSuggestion string       `json:"pivot_suggestion"`
	KillReason      string       `json:"kill_reason"`
}

// validateSynthesis runs the synthesis call to produce the overall GO/PIVOT/KILL verdict.
// Returns the synthResult and LLM cost in USD.
func validateSynthesis(ctx context.Context, c *LLMClient, problem string, claims []ClaimValidation) (synthResult, float64, error) {
	rendered := RenderClaimsForSynthesis(claims)
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultReasonerModel,
		Messages: []Message{
			{Role: "system", Content: validateSynthesisSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(validateSynthesisUserPromptTpl, problem, rendered)},
		},
		MaxTokens:   800,
		Temperature: 0.1,
	})
	if err != nil {
		return synthResult{}, 0, fmt.Errorf("validate synthesis llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var sr synthResult
	if err := json.Unmarshal([]byte(content), &sr); err != nil {
		return synthResult{}, 0, fmt.Errorf("validate synthesis parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}
	return sr, resp.CostUSD, nil
}

// Validate performs adversarial desk-research validation of the top RAT assumptions
// and synthesises a GO / PIVOT / KILL verdict.
func Validate(ctx context.Context, c *LLMClient, frame *FrameResult, h *HypothesisResult) (*ValidationResult, error) {
	top := h.Assumptions
	if len(top) > validateTopN {
		top = top[:validateTopN]
	}

	var totalCost float64
	claims := make([]ClaimValidation, 0, len(top))

	for _, a := range top {
		cv, cost, err := validateClaim(ctx, c, frame.ProblemStatement, a)
		if err != nil {
			return nil, fmt.Errorf("validate claim rank %d: %w", a.RATRank, err)
		}
		claims = append(claims, cv)
		totalCost += cost
	}

	sr, cost, err := validateSynthesis(ctx, c, frame.ProblemStatement, claims)
	if err != nil {
		return nil, fmt.Errorf("validate synthesis: %w", err)
	}
	totalCost += cost

	return &ValidationResult{
		Claims:          claims,
		FinalVerdict:    sr.FinalVerdict,
		VerdictReason:   sr.VerdictReason,
		PivotSuggestion: sr.PivotSuggestion,
		KillReason:      sr.KillReason,
		NeedsExperiment: NeedsExperimentFromClaims(claims),
		CostUSD:         totalCost,
	}, nil
}
```

**Step 4: Run the unit tests to verify they still pass**

```bash
go test ./internal/discovery/... -run "TestNeedsExperiment|TestRenderClaims" -v
```
Expected: PASS.

**Step 5: Build check**

```bash
go build ./...
```
Expected: no errors.

**Step 6: Run integration test (requires API key)**

```bash
go test ./internal/discovery/... -run TestValidate_ProducesVerdict -v -timeout 120s
```
Expected: PASS with logged verdict and cost.

**Step 7: Commit**

```bash
git add internal/discovery/validate.go internal/discovery/validate_test.go
git commit -m "feat: implement Phase 4a Validate() with adversarial per-claim + synthesis LLM calls"
```

---

### Task 3: Artifacts — Session.Validation + writeValidation()

**Files:**
- Modify: `internal/discovery/artifacts.go`
- Modify: `internal/discovery/artifacts_test.go`

**Step 1: Write the failing test**

Add to `internal/discovery/artifacts_test.go`:

```go
func TestArtifacts_WritesValidationFile(t *testing.T) {
	dir := t.TempDir()
	session := &discovery.Session{
		Slug: "test-idea",
		Date: "2026-04-08",
		Frame: &discovery.FrameResult{
			RawIdea:          "test idea",
			ProblemStatement: "test problem",
			Jobs:             []string{"job 1"},
			Appetite:         "small",
		},
		Validation: &discovery.ValidationResult{
			FinalVerdict:  discovery.VerdictGO,
			VerdictReason: "evidence supports both core assumptions",
			Claims: []discovery.ClaimValidation{
				{
					Claim:   "founders lack time for discovery",
					RATRank: 1,
					Verdict: discovery.VerdictSupported,
					Evidence: []discovery.Evidence{
						{Direction: "for", Statement: "62% of indie hackers skip validation", IsEstimate: true},
						{Direction: "against", Statement: "some use customer interviews", IsEstimate: true},
					},
					Confidence: 0.8,
					Notes:      "strong signal from survey data",
				},
			},
			NeedsExperiment: false,
			CostUSD:         0.00123,
		},
	}
	if err := discovery.WriteArtifacts(dir, session); err != nil {
		t.Fatalf("write: %v", err)
	}
	valFile := filepath.Join(dir, "2026-04-08-test-idea-validation.md")
	if _, err := os.Stat(valFile); err != nil {
		t.Errorf("validation file not created: %v", err)
	}
	content, _ := os.ReadFile(valFile)
	s := string(content)
	for _, want := range []string{"GO", "founders lack time", "supported", "evidence supports"} {
		if !strings.Contains(s, want) {
			t.Errorf("validation file missing %q", want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/discovery/... -run TestArtifacts_WritesValidationFile -v
```
Expected: FAIL — `Session` has no field `Validation`.

**Step 3: Update artifacts.go**

In `Session` struct, add `Validation` field after `Scan`:

```go
// Session holds all pipeline state for one discovery run.
type Session struct {
	Slug       string
	Date       string
	Frame      *FrameResult
	Hypothesis *HypothesisResult
	Scan       *ScanResult
	Validation *ValidationResult
}
```

In `WriteArtifacts`, add after the Scan block:

```go
	if s.Validation != nil {
		if err := writeValidation(prefix+"-validation.md", s.Validation); err != nil {
			return err
		}
	}
```

Add the new function at the bottom of artifacts.go:

```go
func writeValidation(path string, v *ValidationResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Validation\n\n")

	// Verdict banner
	verdictIcon := map[FinalVerdict]string{
		VerdictGO:    "✅",
		VerdictPIVOT: "🔄",
		VerdictKILL:  "❌",
	}
	icon := verdictIcon[v.FinalVerdict]
	if icon == "" {
		icon = "❓"
	}
	fmt.Fprintf(&b, "## %s Final Verdict: %s\n\n", icon, v.FinalVerdict)
	fmt.Fprintf(&b, "%s\n\n", v.VerdictReason)

	if v.PivotSuggestion != "" {
		fmt.Fprintf(&b, "**Pivot suggestion:** %s\n\n", v.PivotSuggestion)
	}
	if v.KillReason != "" {
		fmt.Fprintf(&b, "**Kill reason:** %s\n\n", v.KillReason)
	}
	if v.NeedsExperiment {
		fmt.Fprintf(&b, "> ⚠️ **Needs experiment:** one or more claims have insufficient desk-research data — Phase 4b recommended.\n\n")
	}

	fmt.Fprintf(&b, "## Claim Validation\n\n")
	for _, c := range v.Claims {
		verdictStr := strings.ToUpper(string(c.Verdict))
		fmt.Fprintf(&b, "### Rank %d — %s (confidence %.0f%%)\n\n", c.RATRank, verdictStr, c.Confidence*100)
		fmt.Fprintf(&b, "**Claim:** %s\n\n", c.Claim)
		if c.Notes != "" {
			fmt.Fprintf(&b, "**Notes:** %s\n\n", c.Notes)
		}
		if len(c.Evidence) > 0 {
			fmt.Fprintf(&b, "| Direction | Evidence | Estimate? |\n")
			fmt.Fprintf(&b, "|-----------|----------|-----------|\n")
			for _, e := range c.Evidence {
				est := "no"
				if e.IsEstimate {
					est = "yes"
				}
				stmt := e.Statement
				if e.SourceURL != "" {
					stmt = fmt.Sprintf("[%s](%s)", e.Statement, e.SourceURL)
				}
				fmt.Fprintf(&b, "| %s | %s | %s |\n", strings.ToUpper(e.Direction), stmt, est)
			}
			fmt.Fprintf(&b, "\n")
		}
	}

	fmt.Fprintf(&b, "---\n\n*Cost: $%.5f*\n", v.CostUSD)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
```

Note: `writeValidation` references `FinalVerdict`, `VerdictGO`, etc. from `validate.go` — that's fine since both files are in the same package. Also uses `strings.ToUpper(string(c.Verdict))` — `c.Verdict` is `ClaimVerdict` (a `string` type), casting to `string` is necessary.

**Step 4: Run test to verify it passes**

```bash
go test ./internal/discovery/... -run TestArtifacts_WritesValidationFile -v
```
Expected: PASS.

**Step 5: Run all artifact tests**

```bash
go test ./internal/discovery/... -run TestArtifacts -v
```
Expected: all PASS.

**Step 6: Build check**

```bash
go build ./...
```
Expected: no errors.

**Step 7: Commit**

```bash
git add internal/discovery/artifacts.go internal/discovery/artifacts_test.go
git commit -m "feat: Session.Validation field + writeValidation() artifact"
```

---

### Task 4: CLI — Phase 4a in cmd_discover.go

**Files:**
- Modify: `cmd/sdp/cmd_discover.go`

**Step 1: Read the file before editing**

Open `cmd/sdp/cmd_discover.go` and note line numbers to understand where to insert Phase 4a (after Checkpoint C / artifacts write, before beads issue creation). Specifically, add after the `discovery.WriteArtifacts` block and before the beads section.

Actually, Phase 4a should run BEFORE writing artifacts so the validation is included in the artifact write. The insertion point is after Phase 3 SCAN + Checkpoint C print, before `WriteArtifacts`.

The current pipeline in `runDiscover`:
1. Phase 1: FRAME
2. Checkpoint A: clarifications
3. Phase 2: HYPOTHESIZE
4. Checkpoint B: hypothesis summary
5. Phase 3: SCAN
6. Checkpoint C: `RenderCheckpoint`
7. `WriteArtifacts`
8. Beads issue

Insert Phase 4a between step 6 and step 7.

**Step 2: Add Phase 4a block**

After the line `fmt.Println(discovery.RenderCheckpoint(scanResult))` and before the `WriteArtifacts` call, insert:

```go
	// ── Phase 4a: VALIDATE (desk research) ────────────────────────
	fmt.Printf("🔬 Phase 4a: Validating top assumptions...\n")
	validation, err := discovery.Validate(ctx, client, frame, hypothesis)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: validate: %v\n", err)
		os.Exit(1)
	}
	session.Validation = validation

	// Print verdict prominently
	verdictIcon := map[discovery.FinalVerdict]string{
		discovery.VerdictGO:    "✅ GO",
		discovery.VerdictPIVOT: "🔄 PIVOT",
		discovery.VerdictKILL:  "❌ KILL",
	}
	verdictLabel := verdictIcon[validation.FinalVerdict]
	if verdictLabel == "" {
		verdictLabel = string(validation.FinalVerdict)
	}
	fmt.Printf("\n── Checkpoint D — Validation Verdict ──\n\n")
	fmt.Printf("  Verdict:  %s\n", verdictLabel)
	fmt.Printf("  Reason:   %s\n", validation.VerdictReason)
	if validation.PivotSuggestion != "" {
		fmt.Printf("  Pivot to: %s\n", validation.PivotSuggestion)
	}
	if validation.KillReason != "" {
		fmt.Printf("  Kill why: %s\n", validation.KillReason)
	}
	fmt.Printf("  Claims:   %d validated (needs_experiment=%v)\n",
		len(validation.Claims), validation.NeedsExperiment)
	fmt.Printf("  Cost:     $%.5f\n\n", validation.CostUSD)
```

The `verdictIcon` map uses `discovery.FinalVerdict` keys. This compiles fine since `FinalVerdict` is an exported string type.

**Step 3: Build check**

```bash
go build ./...
```
Expected: no errors.

**Step 4: Quick smoke test (dry-run check)**

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go vet ./...
```
Expected: no warnings.

**Step 5: Commit**

```bash
git add cmd/sdp/cmd_discover.go
git commit -m "feat: Phase 4a VALIDATE in discover pipeline — Checkpoint D GO/PIVOT/KILL verdict"
```

---

### Task 5: Dogfood — 3 ideas end-to-end

**Goal:** Run `sdp discover` on 3 real ideas, review validation output quality, calibrate prompts if needed.

**Step 1: Build the binary**

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go build -o bin/sdp ./cmd/sdp
```

**Step 2: Run idea 1**

```bash
./bin/sdp discover "automate product discovery using AI agents"
```

Review output:
- Does each claim have ≥3 evidence items each direction?
- Is the verdict (GO/PIVOT/KILL) defensible?
- Are `source_url` fields empty (not hallucinated)?
- Does `is_estimate: true` appear on most evidence items?
- Does `validation.md` render cleanly?

**Step 3: Run idea 2 — a riskier idea to trigger KILL or PIVOT**

```bash
./bin/sdp discover "build a crypto trading bot for retail investors"
```

Review: expect KILL or PIVOT (high regulatory/market risk). Verify verdict reason is specific.

**Step 4: Run idea 3 — a consumer app idea**

```bash
./bin/sdp discover "AI meal planner that learns family preferences"
```

Review: verdict should reflect real market saturation signals (many competitors).

**Step 5: Calibration — if prompt quality is poor**

Common issues and fixes:

| Symptom | Fix |
|---------|-----|
| `source_url` has hallucinated URLs (e.g. `https://somesurvey.com/data`) | Add to system prompt: "If you are not 100% certain a URL is real and publicly accessible, set source_url to empty string." |
| All evidence `is_estimate: true` even for well-known data | Relax: acceptable behaviour; is_estimate guards against false precision |
| Claim verdict always `insufficient_data` | Strengthen prompt: add examples of what counts as "supported" vs "insufficient_data" |
| Synthesis calls incorrect GO when rank-1 is contradicted | Tighten synthesis rules: "If rank-1 claim verdict is 'contradicted' with confidence >= 0.7, verdict MUST be KILL or PIVOT" |
| JSON truncation error on claim | Bump MaxTokens from 1500 to 2000 in `validateClaim` |

To apply a fix, edit `internal/discovery/validate.go`, rebuild, and re-run the failing idea.

**Step 6: Commit calibration changes (if any)**

```bash
git add internal/discovery/validate.go
git commit -m "calibrate: improve validate prompt quality from dogfood run"
```

**Step 7: Push**

```bash
git push
```

---

## Summary of files changed

| File | Action |
|------|--------|
| `internal/discovery/validate.go` | **Create** — types, helpers, LLM calls, Validate() |
| `internal/discovery/validate_test.go` | **Create** — unit tests + integration test |
| `internal/discovery/artifacts.go` | **Modify** — Session.Validation, writeValidation() |
| `internal/discovery/artifacts_test.go` | **Modify** — TestArtifacts_WritesValidationFile |
| `cmd/sdp/cmd_discover.go` | **Modify** — Phase 4a + Checkpoint D |

## Key invariants

- `Validate()` always calls `DefaultReasonerModel` (never planner/synth model)
- Evidence `source_url` is empty string if URL unknown — never fabricated
- `NeedsExperiment` is a pure function of `ClaimVerdict` values (tested without LLM)
- `writeValidation()` renders `PivotSuggestion` only when non-empty (same for `KillReason`)
- Top N = 3 assumptions; constant `validateTopN` controls this
- `RenderClaimsForSynthesis` is exported so it can be unit-tested

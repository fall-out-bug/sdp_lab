# Discovery Phase 2: HYPOTHESIZE Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Phase 2 HYPOTHESIZE to `sdp discover` — takes the FrameResult and produces a Strategyzer Test Card, RAT-ranked assumptions, and functional requirements; insert Checkpoint A (typed clarifications) between FRAME and HYPOTHESIZE.

**Architecture:** Two new files in `internal/discovery/` (`hypothesize.go`, `clarify.go`), updates to `artifacts.go` (Session gains `Hypothesis` field, writes `hypothesis.md`), and updates to `cmd/sdp/cmd_discover.go` (pipeline becomes FRAME → Checkpoint A → HYPOTHESIZE → Checkpoint B → SCAN → Checkpoint C). All LLM calls use `DefaultPlannerModel` (deepseek/deepseek-v3.2). RAT scores are computed locally from LLM-supplied risk/uncertainty levels, not by the LLM.

**Tech Stack:** Go 1.26, `sort` stdlib, existing `internal/discovery` package, OpenRouter API via `LLMClient`.

---

## Task 1: Hypothesis types + `Hypothesize()` function

**Files:**
- Create: `internal/discovery/hypothesize.go`
- Create: `internal/discovery/hypothesize_test.go`

### Step 1: Write the failing test

```go
// internal/discovery/hypothesize_test.go
package discovery_test

import (
	"context"
	"os"
	"testing"
	"sdp_dev/internal/discovery"
)

func TestHypothesizeProducesTestCard(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
	frame := &discovery.FrameResult{
		ProblemStatement: "developers spend hours on manual product discovery before writing specs",
		Jobs:             []string{"developer wants to validate ideas quickly before investing in implementation"},
		Appetite:         "medium",
		RawIdea:          "automate product discovery using AI agents",
	}
	result, err := discovery.Hypothesize(context.Background(), c, frame)
	if err != nil {
		t.Fatalf("hypothesize: %v", err)
	}
	if result.WeBelieve == "" {
		t.Error("empty we_believe")
	}
	if result.ToVerify == "" {
		t.Error("empty to_verify")
	}
	if result.WeMeasure == "" {
		t.Error("empty we_measure")
	}
	if result.WeAreRightIf == "" {
		t.Error("empty we_are_right_if")
	}
	if len(result.Assumptions) == 0 {
		t.Error("no assumptions")
	}
	// verify RAT ranking: rank 1 has highest or equal RAT score
	if len(result.Assumptions) > 1 {
		if result.Assumptions[0].RATScore < result.Assumptions[len(result.Assumptions)-1].RATScore {
			t.Error("assumptions not sorted by RAT score descending")
		}
		if result.Assumptions[0].RATRank != 1 {
			t.Errorf("first assumption should have RATRank=1, got %d", result.Assumptions[0].RATRank)
		}
	}
	t.Logf("hypothesis: we_believe=%s", result.WeBelieve)
	t.Logf("riskiest assumption: %s (score=%.0f)", result.Assumptions[0].Statement, result.Assumptions[0].RATScore)
}
```

### Step 2: Run to confirm failure

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... -run TestHypothesizeProducesTestCard -v 2>&1 | head -5
```

Expected: `FAIL — discovery.Hypothesize undefined`

### Step 3: Implement `internal/discovery/hypothesize.go`

```go
// internal/discovery/hypothesize.go
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Assumption is one hypothesis assumption with RAT (Riskiest Assumption Test) metadata.
type Assumption struct {
	Statement   string  `json:"statement"`
	RiskLevel   string  `json:"risk_level"`   // high|medium|low — impact if wrong
	Uncertainty string  `json:"uncertainty"`  // high|medium|low — how unknown
	RATScore    float64 `json:"rat_score"`    // computed: risk_val × uncertainty_val (1–9)
	RATRank     int     `json:"rat_rank"`     // 1 = riskiest; assigned after sort
}

// HypothesisResult is the output of Phase 2 HYPOTHESIZE.
type HypothesisResult struct {
	WeBelieve    string       `json:"we_believe"`     // Strategyzer Test Card: belief statement
	ToVerify     string       `json:"to_verify"`      // cheapest test
	WeMeasure    string       `json:"we_measure"`     // key metric
	WeAreRightIf string       `json:"we_are_right_if"` // success criterion
	Assumptions  []Assumption `json:"assumptions"`    // RAT-ranked, index 0 = riskiest
	Requirements []string     `json:"requirements"`   // functional requirements
	RawIdea      string       `json:"raw_idea"`
}

const hypothesizeSystemPrompt = `You are a product hypothesis agent specializing in Strategyzer Test Cards and assumption mapping.
Respond ONLY with valid JSON — no markdown, no explanation.`

const hypothesizeUserPromptTpl = `Generate a product hypothesis for this problem using the Strategyzer Test Card format.

PROBLEM: %s
JOBS: %s
APPETITE: %s

Return JSON with this exact schema:
{"we_believe":"customer segment needs to [job] because [reason]","to_verify":"cheapest test to validate the core assumption","we_measure":"the key metric","we_are_right_if":"measurable success criterion (e.g. >50 signups in 14 days)","assumptions":[{"statement":"assumption that must be true for the hypothesis to hold","risk_level":"high|medium|low","uncertainty":"high|medium|low"}],"requirements":["functional requirement 1","functional requirement 2"]}`

// Hypothesize generates a Strategyzer Test Card and RAT-ranked assumptions from a FrameResult.
func Hypothesize(ctx context.Context, c *LLMClient, frame *FrameResult) (*HypothesisResult, error) {
	jobs := strings.Join(frame.Jobs, "; ")
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultPlannerModel,
		Messages: []Message{
			{Role: "system", Content: hypothesizeSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(hypothesizeUserPromptTpl,
				frame.ProblemStatement, jobs, frame.Appetite)},
		},
		MaxTokens:   1200,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("hypothesize llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	// LLM returns assumptions without scores — parse raw first
	var raw struct {
		WeBelieve    string `json:"we_believe"`
		ToVerify     string `json:"to_verify"`
		WeMeasure    string `json:"we_measure"`
		WeAreRightIf string `json:"we_are_right_if"`
		Assumptions  []struct {
			Statement   string `json:"statement"`
			RiskLevel   string `json:"risk_level"`
			Uncertainty string `json:"uncertainty"`
		} `json:"assumptions"`
		Requirements []string `json:"requirements"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("hypothesize parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}

	// Compute RAT scores and rank
	assumptions := make([]Assumption, len(raw.Assumptions))
	for i, a := range raw.Assumptions {
		assumptions[i] = Assumption{
			Statement:   a.Statement,
			RiskLevel:   a.RiskLevel,
			Uncertainty: a.Uncertainty,
			RATScore:    ratScore(a.RiskLevel, a.Uncertainty),
		}
	}
	assumptions = computeRATRanks(assumptions)

	result := &HypothesisResult{
		WeBelieve:    raw.WeBelieve,
		ToVerify:     raw.ToVerify,
		WeMeasure:    raw.WeMeasure,
		WeAreRightIf: raw.WeAreRightIf,
		Assumptions:  assumptions,
		Requirements: raw.Requirements,
		RawIdea:      frame.RawIdea,
	}
	return result, nil
}

// ratScore converts risk and uncertainty text levels to a numeric RAT score (1–9).
func ratScore(riskLevel, uncertainty string) float64 {
	val := map[string]float64{"high": 3, "medium": 2, "low": 1}
	r := val[riskLevel]
	u := val[uncertainty]
	if r == 0 {
		r = 1 // default to low for unknown values
	}
	if u == 0 {
		u = 1
	}
	return r * u
}

// computeRATRanks sorts assumptions by RAT score descending and assigns RATRank (1 = riskiest).
func computeRATRanks(assumptions []Assumption) []Assumption {
	sorted := make([]Assumption, len(assumptions))
	copy(sorted, assumptions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RATScore > sorted[j].RATScore
	})
	for i := range sorted {
		sorted[i].RATRank = i + 1
	}
	return sorted
}
```

### Step 4: Run the test

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
export $(grep -v '^#' .env | xargs) 2>/dev/null
go test ./internal/discovery/... -run TestHypothesizeProducesTestCard -v -timeout 45s
```

Expected: PASS — WeBelieve non-empty, assumptions sorted by RAT score descending.

### Step 5: Run full package suite

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... 2>&1 | tail -3
```

Expected: `ok sdp_dev/internal/discovery`

### Step 6: Commit

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
git add internal/discovery/hypothesize.go internal/discovery/hypothesize_test.go
git commit -m "feat(discovery): Phase 2 HYPOTHESIZE — Test Card + RAT ranking"
```

---

## Task 2: Typed clarification requests

**Files:**
- Create: `internal/discovery/clarify.go`
- Create: `internal/discovery/clarify_test.go`

### Step 1: Write the failing test

```go
// internal/discovery/clarify_test.go
package discovery_test

import (
	"context"
	"os"
	"testing"
	"sdp_dev/internal/discovery"
)

func TestGenerateClarifications_ProducesQuestions(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
	frame := &discovery.FrameResult{
		ProblemStatement: "developers spend hours on manual product discovery before writing specs",
		Jobs:             []string{"developer wants to validate ideas quickly before investing in implementation"},
		Appetite:         "medium",
		RawIdea:          "automate product discovery using AI agents",
	}
	reqs, err := discovery.GenerateClarifications(context.Background(), c, frame)
	if err != nil {
		t.Fatalf("clarify: %v", err)
	}
	if len(reqs) == 0 {
		t.Error("no clarification requests generated")
	}
	for i, r := range reqs {
		if r.Question == "" {
			t.Errorf("clarification[%d]: empty question", i)
		}
		validTypes := map[string]bool{
			"missing_info": true, "ambiguous_requirement": true,
			"approach_choice": true, "risk_confirmation": true,
		}
		if !validTypes[string(r.Type)] {
			t.Errorf("clarification[%d]: unknown type %q", i, r.Type)
		}
	}
	t.Logf("clarifications: %d requests", len(reqs))
	for _, r := range reqs {
		t.Logf("  [%s] %s", r.Type, r.Question)
	}
}
```

### Step 2: Run to confirm failure

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... -run TestGenerateClarifications -v 2>&1 | head -5
```

Expected: `FAIL — discovery.GenerateClarifications undefined`

### Step 3: Implement `internal/discovery/clarify.go`

```go
// internal/discovery/clarify.go
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ClarificationType is the category of a clarification request (from DeerFlow INSPIRE).
type ClarificationType string

const (
	ClarifyMissingInfo          ClarificationType = "missing_info"
	ClarifyAmbiguousRequirement ClarificationType = "ambiguous_requirement"
	ClarifyApproachChoice       ClarificationType = "approach_choice"
	ClarifyRiskConfirmation     ClarificationType = "risk_confirmation"
)

// ClarificationRequest is a typed question for the human before hypothesis generation.
type ClarificationRequest struct {
	Type     ClarificationType `json:"type"`
	Question string            `json:"question"`
	Context  string            `json:"context"`           // why this question matters
	Options  []string          `json:"options,omitempty"` // only for approach_choice
}

const clarifySystemPrompt = `You are a product discovery agent that identifies gaps in problem framing.
Respond ONLY with valid JSON — no markdown, no explanation.`

const clarifyUserPromptTpl = `Identify 2–3 clarifying questions that would materially improve the product hypothesis for this problem.

PROBLEM: %s
JOBS: %s

Use these types:
- missing_info: key data or context we don't have
- ambiguous_requirement: something in the problem statement that could mean multiple things
- approach_choice: a fork where the answer changes the design significantly
- risk_confirmation: a high-stakes assumption that needs human validation

Return JSON:
{"clarifications":[{"type":"missing_info|ambiguous_requirement|approach_choice|risk_confirmation","question":"specific, answerable question","context":"why this matters for the hypothesis","options":["option A","option B"]}]}`

// GenerateClarifications produces typed clarification questions for the human before hypothesis generation.
func GenerateClarifications(ctx context.Context, c *LLMClient, frame *FrameResult) ([]ClarificationRequest, error) {
	jobs := strings.Join(frame.Jobs, "; ")
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultPlannerModel,
		Messages: []Message{
			{Role: "system", Content: clarifySystemPrompt},
			{Role: "user", Content: fmt.Sprintf(clarifyUserPromptTpl,
				frame.ProblemStatement, jobs)},
		},
		MaxTokens:   600,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("clarify llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var raw struct {
		Clarifications []ClarificationRequest `json:"clarifications"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("clarify parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}
	return raw.Clarifications, nil
}
```

### Step 4: Run the test

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
export $(grep -v '^#' .env | xargs) 2>/dev/null
go test ./internal/discovery/... -run TestGenerateClarifications -v -timeout 30s
```

Expected: PASS — 2–3 clarification requests with valid types.

### Step 5: Run full package suite

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... 2>&1 | tail -3
```

### Step 6: Commit

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
git add internal/discovery/clarify.go internal/discovery/clarify_test.go
git commit -m "feat(discovery): typed clarification requests — Checkpoint A"
```

---

## Task 3: Update artifact writer for hypothesis

**Files:**
- Modify: `internal/discovery/artifacts.go`
- Modify: `internal/discovery/artifacts_test.go`

### Step 1: Read the current artifacts.go

The `Session` struct (lines 13–18) currently has `Frame` and `Scan`. The `WriteArtifacts` function (lines 48–65) writes frame and scan files.

### Step 2: Update `artifacts_test.go` to include hypothesis

Add this test (do NOT delete the existing `TestArtifacts_WritesFiles`):

```go
func TestArtifacts_WritesHypothesisFile(t *testing.T) {
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
		Hypothesis: &discovery.HypothesisResult{
			WeBelieve:    "developers need automated discovery",
			ToVerify:     "run a landing page test",
			WeMeasure:    "signups in 14 days",
			WeAreRightIf: ">50 signups",
			Assumptions: []discovery.Assumption{
				{Statement: "gap exists", RiskLevel: "high", Uncertainty: "high", RATScore: 9, RATRank: 1},
			},
			Requirements: []string{"CLI entry point", "markdown output"},
			RawIdea:      "test idea",
		},
	}
	if err := discovery.WriteArtifacts(dir, session); err != nil {
		t.Fatalf("write: %v", err)
	}
	hypothesisFile := filepath.Join(dir, "2026-04-08-test-idea-hypothesis.md")
	if _, err := os.Stat(hypothesisFile); err != nil {
		t.Errorf("hypothesis file not created: %v", err)
	}
	content, _ := os.ReadFile(hypothesisFile)
	if !strings.Contains(string(content), "developers need automated discovery") {
		t.Error("hypothesis file missing we_believe")
	}
	if !strings.Contains(string(content), "gap exists") {
		t.Error("hypothesis file missing assumption")
	}
}
```

Run to confirm it fails:
```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... -run TestArtifacts_WritesHypothesisFile -v 2>&1 | head -8
```

Expected: compile error — `Session` has no field `Hypothesis`

### Step 3: Update `internal/discovery/artifacts.go`

**3a.** Add `Hypothesis *HypothesisResult` to Session (after line 17, before closing brace):

The Session struct becomes:
```go
type Session struct {
	Slug       string
	Date       string
	Frame      *FrameResult
	Hypothesis *HypothesisResult
	Scan       *ScanResult
}
```

**3b.** Add hypothesis write call in `WriteArtifacts` (after the Frame block, before Scan):
```go
	if s.Hypothesis != nil {
		if err := writeHypothesis(prefix+"-hypothesis.md", s.Hypothesis); err != nil {
			return err
		}
	}
```

**3c.** Add `writeHypothesis` function at the end of artifacts.go:

```go
func writeHypothesis(path string, h *HypothesisResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Hypothesis\n\n")
	fmt.Fprintf(&b, "**Raw idea:** %s\n\n", h.RawIdea)
	fmt.Fprintf(&b, "## Test Card (Strategyzer)\n\n")
	fmt.Fprintf(&b, "**We believe** %s\n\n", h.WeBelieve)
	fmt.Fprintf(&b, "**To verify this**, we will %s\n\n", h.ToVerify)
	fmt.Fprintf(&b, "**We'll measure** %s\n\n", h.WeMeasure)
	fmt.Fprintf(&b, "**We are right if** %s\n\n", h.WeAreRightIf)
	fmt.Fprintf(&b, "## Assumptions (RAT-Ranked)\n\n")
	fmt.Fprintf(&b, "| Rank | Assumption | Risk | Uncertainty | RAT Score |\n")
	fmt.Fprintf(&b, "|------|-----------|------|-------------|----------|\n")
	for _, a := range h.Assumptions {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %.0f |\n",
			a.RATRank, a.Statement, a.RiskLevel, a.Uncertainty, a.RATScore)
	}
	if len(h.Assumptions) > 0 {
		fmt.Fprintf(&b, "\n**Riskiest assumption (rank 1):** %s\n", h.Assumptions[0].Statement)
	}
	if len(h.Requirements) > 0 {
		fmt.Fprintf(&b, "\n## Requirements\n\n")
		for _, r := range h.Requirements {
			fmt.Fprintf(&b, "- %s\n", r)
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
```

### Step 4: Run the new test

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... -run TestArtifacts -v
```

Expected: both `TestArtifacts_WritesFiles` and `TestArtifacts_WritesHypothesisFile` PASS.

### Step 5: Run full package suite

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... 2>&1 | tail -3
```

Expected: `ok sdp_dev/internal/discovery`

### Step 6: Commit

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
git add internal/discovery/artifacts.go internal/discovery/artifacts_test.go
git commit -m "feat(discovery): artifact writer — hypothesis.md output"
```

---

## Task 4: Wire Phase 2 into `sdp discover`

**Files:**
- Modify: `cmd/sdp/cmd_discover.go`

The current pipeline in `runDiscover` is: FRAME → SCAN → checkpoint → artifacts → beads.
The new pipeline is: FRAME → Checkpoint A → HYPOTHESIZE → Checkpoint B summary → SCAN → Checkpoint C → artifacts → beads.

### Step 1: No test needed (CLI integration — verified by build + smoke test)

### Step 2: Update `cmd/sdp/cmd_discover.go`

Replace the section between `session.Frame = frame` (line 49) and the SCAN section (line 53) with the Checkpoint A + Phase 2 block. Also update the beads issue to include hypothesis data.

The full updated `runDiscover` body (keep the function signature and flag setup unchanged, replace from `session.Frame = frame` onwards):

```go
	session.Frame = frame
	fmt.Printf("   problem:  %s\n", frame.ProblemStatement)
	fmt.Printf("   appetite: %s\n\n", frame.Appetite)

	// ── Checkpoint A: Typed clarifications (non-blocking) ──────────
	fmt.Printf("💬 Checkpoint A: Generating clarifications...\n")
	clarifications, err := discovery.GenerateClarifications(ctx, client, frame)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   warning: clarifications: %v\n", err)
	} else if len(clarifications) > 0 {
		fmt.Printf("\n── Clarifications (refine idea before continuing) ──\n\n")
		for i, c := range clarifications {
			fmt.Printf("  %d. [%s] %s\n", i+1, c.Type, c.Question)
			if c.Context != "" {
				fmt.Printf("     context: %s\n", c.Context)
			}
			if len(c.Options) > 0 {
				for j, opt := range c.Options {
					fmt.Printf("     [%c] %s\n", 'A'+j, opt)
				}
			}
		}
		fmt.Printf("\n   (proceeding with defaults — re-run with refined idea to update)\n\n")
	}

	// ── Phase 2: HYPOTHESIZE ────────────────────────────────────────
	fmt.Printf("💡 Phase 2: Generating hypothesis...\n")
	hypothesis, err := discovery.Hypothesize(ctx, client, frame)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: hypothesize: %v\n", err)
		os.Exit(1)
	}
	session.Hypothesis = hypothesis
	fmt.Printf("   we believe: %s\n", hypothesis.WeBelieve)
	if len(hypothesis.Assumptions) > 0 {
		fmt.Printf("   riskiest:   %s (RAT=%.0f)\n",
			hypothesis.Assumptions[0].Statement,
			hypothesis.Assumptions[0].RATScore)
	}
	fmt.Printf("\n")

	// ── Checkpoint B: Hypothesis summary ───────────────────────────
	fmt.Printf("── Checkpoint B — Hypothesis ──\n\n")
	fmt.Printf("  Test Card:\n")
	fmt.Printf("    We believe: %s\n", hypothesis.WeBelieve)
	fmt.Printf("    To verify:  %s\n", hypothesis.ToVerify)
	fmt.Printf("    Measure:    %s\n", hypothesis.WeMeasure)
	fmt.Printf("    Right if:   %s\n\n", hypothesis.WeAreRightIf)
	if len(hypothesis.Assumptions) > 0 {
		fmt.Printf("  RAT-ranked assumptions:\n")
		for _, a := range hypothesis.Assumptions {
			fmt.Printf("    %d. [RAT=%.0f] %s (%s risk, %s uncertainty)\n",
				a.RATRank, a.RATScore, a.Statement, a.RiskLevel, a.Uncertainty)
		}
		fmt.Printf("\n")
	}
```

Keep the SCAN section and everything after it unchanged EXCEPT update `createDiscoveryIssue` call to pass hypothesis:

Change the beads creation call:
```go
	issueID, err := createDiscoveryIssue(idea, frame, hypothesis, absOut)
```

And update `createDiscoveryIssue` signature and body:
```go
func createDiscoveryIssue(idea string, frame *discovery.FrameResult,
	hypothesis *discovery.HypothesisResult, artifactDir string) (string, error) {

	hypoSection := ""
	if hypothesis != nil {
		hypoSection = fmt.Sprintf("\n\n**Hypothesis:** %s\n\n**Riskiest assumption:** %s",
			hypothesis.WeBelieve,
			func() string {
				if len(hypothesis.Assumptions) > 0 {
					return hypothesis.Assumptions[0].Statement
				}
				return "—"
			}())
	}

	desc := fmt.Sprintf(
		"## Discovery: %s\n\n**Problem:** %s\n\n**Appetite:** %s%s\n\n**Artifacts:** %s/",
		idea, frame.ProblemStatement, frame.Appetite, hypoSection, artifactDir,
	)
	cmd := exec.Command("bd", "create",
		"--title=Discovery: "+idea,
		"--description="+desc,
		"--type=task",
		"--priority=2",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bd create: %w: %s", err, out)
	}
	return string(out), nil
}
```

### Step 3: Build

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go build -o sdp_bin ./cmd/sdp/ 2>&1
```

Expected: success.

### Step 4: Smoke test (usage only, no API key needed)

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
./sdp_bin discover 2>&1
```

Expected: `usage: sdp discover [--out DIR] [--model MODEL] "raw idea"`

### Step 5: Commit

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
git add cmd/sdp/cmd_discover.go
git commit -m "feat(discovery): wire Phase 2 HYPOTHESIZE + Checkpoint A into sdp discover"
```

---

## Task 5: Dogfood — full pipeline on 3 ideas

**Goal:** Validate Phase 2 quality (Test Card coherence, RAT ranking accuracy) and Checkpoint A usefulness before building Phase 4.

### Step 1: Build

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go build -o sdp_bin ./cmd/sdp/
```

### Step 2: Run on 3 ideas

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
export $(grep -v '^#' .env | xargs) 2>/dev/null

./sdp_bin discover "automate product discovery using AI agents" 2>&1
echo "==="
./sdp_bin discover "build a CLI tool that monitors competitor pricing in real time" 2>&1
echo "==="
./sdp_bin discover "replace standups with async AI-generated team summaries" 2>&1
```

### Step 3: Evaluate quality

For each run, assess:
1. **Test Card coherence** — Does `we_believe` describe a real customer need? Is `to_verify` actually the cheapest test? Is `we_are_right_if` measurable?
2. **RAT ranking accuracy** — Is rank-1 (highest RAT score) genuinely the riskiest assumption? Are `risk_level` and `uncertainty` calibrated?
3. **Checkpoint A usefulness** — Are the clarification questions specific and actionable? Do they address real gaps in the framing?
4. **Requirements quality** — Are they functional (what the system must do) not technical (how it does it)?

### Step 4: Check artifact files

```bash
ls /Users/fall_out_bug/projects/vibe_coding/sdp_lab/docs/discovery/*hypothesis* 2>/dev/null
```

Expected: 3 hypothesis files created.

### Step 5: Calibrate if needed

If Test Card quality is poor (e.g., `we_believe` is too generic, RAT scores all the same), adjust the prompt in `hypothesize.go`:
- Add a concrete example in the prompt
- Increase MaxTokens to 1500

If clarifications are irrelevant or too generic, adjust `clarify.go` prompt:
- Make it more specific: "identify only questions that would change the hypothesis significantly"

If adjustments made, re-run tests and commit:
```bash
go test ./internal/discovery/... 2>&1 | tail -3
git add internal/discovery/hypothesize.go internal/discovery/clarify.go
git commit -m "fix(discovery): calibrate Phase 2 prompts from dogfood"
```

---

## What's NOT in this plan (next iterations)

- Phase 4a: Desk research validation (claims → evidence for/against)
- Phase 4b: Experiment design and brief
- Interactive Checkpoint A (stdin prompt with TTY detection)
- Hypothesis refinement loop (re-run hypothesize with clarification answers)
- MCP fan-out in Phase 3 (Exa, Brave, GitHub)
- PIVOT flow (back to Phase 2 with new inputs after scan)

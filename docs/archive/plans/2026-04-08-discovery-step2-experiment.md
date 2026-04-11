# Discovery Step 2: Phase 4b EXPERIMENT

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** When Phase 4a returns `NeedsExperiment=true` (one or more assumptions have `insufficient_data`), run Phase 4b to generate an `ExperimentBrief` — the cheapest test design to resolve unvalidated assumptions — and create a child beads issue for it.

**Architecture:** Three-component addition: (1) `internal/discovery/experiment.go` owns types + `GenerateExperiment()` LLM call (uses `DefaultReasonerModel`), (2) `artifacts.go` gains `Session.Experiment` and `writeExperiment()`, (3) `cmd_discover.go` calls Phase 4b conditionally after Phase 4a and creates a child beads issue. Phase 4b only runs when `validation.NeedsExperiment == true` — it is always skipped on clean GO/KILL verdicts where all claims are settled.

**Tech Stack:** Go 1.26, `internal/discovery` package, OpenRouter API via `LLMClient`, `DefaultReasonerModel = "openai/gpt-5.4-mini"`, `encoding/json`, `strings`, `fmt`.

---

### Task 1: ExperimentBrief types + pure helpers

**Files:**
- Create: `internal/discovery/experiment.go`
- Create: `internal/discovery/experiment_test.go`

**Step 1: Write the failing test**

```go
// internal/discovery/experiment_test.go
package discovery_test

import (
	"context"
	"os"
	"testing"

	"sdp_dev/internal/discovery"
)

func TestExperimentFormat_Constants(t *testing.T) {
	formats := []discovery.ExperimentFormat{
		discovery.ExperimentSmokeTest,
		discovery.ExperimentLandingPage,
		discovery.ExperimentCustomerInterview,
		discovery.ExperimentWizardOfOz,
	}
	for _, f := range formats {
		if string(f) == "" {
			t.Errorf("empty format constant: %v", f)
		}
	}
}

func TestGenerateExperiment_SkipWithoutAPIKey(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
	frame := &discovery.FrameResult{
		ProblemStatement: "developers spend hours on manual product discovery",
		Jobs:             []string{"validate ideas cheaply"},
		Appetite:         "medium",
	}
	val := &discovery.ValidationResult{
		FinalVerdict:    discovery.VerdictPIVOT,
		NeedsExperiment: true,
		Claims: []discovery.ClaimValidation{
			{
				Claim:   "LLM-generated validation is trusted by founders",
				RATRank: 1,
				Verdict: discovery.VerdictInsufficientData,
				Notes:   "no strong evidence either way",
			},
		},
	}
	brief, err := discovery.GenerateExperiment(context.Background(), c, frame, val)
	if err != nil {
		t.Fatalf("GenerateExperiment: %v", err)
	}
	if brief.Format == "" {
		t.Error("empty experiment format")
	}
	if brief.Objective == "" {
		t.Error("empty objective")
	}
	if brief.SuccessMetric == "" {
		t.Error("empty success metric")
	}
	if brief.TimeBoxDays <= 0 {
		t.Error("time_box_days must be positive")
	}
	if len(brief.SetupSteps) == 0 {
		t.Error("no setup steps")
	}
	validFormats := map[discovery.ExperimentFormat]bool{
		discovery.ExperimentSmokeTest:         true,
		discovery.ExperimentLandingPage:        true,
		discovery.ExperimentCustomerInterview:  true,
		discovery.ExperimentWizardOfOz:         true,
	}
	if !validFormats[brief.Format] {
		t.Errorf("invalid experiment format: %q", brief.Format)
	}
	t.Logf("format: %s, time_box: %d days, cost: $%.5f", brief.Format, brief.TimeBoxDays, brief.CostUSD)
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/discovery/... -run "TestExperimentFormat_Constants" -v
```
Expected: FAIL — `discovery.ExperimentFormat` undefined.

**Step 3: Write minimal implementation (types only + stub)**

```go
// internal/discovery/experiment.go
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ExperimentFormat is the type of cheapest experiment to run.
type ExperimentFormat string

const (
	ExperimentSmokeTest        ExperimentFormat = "smoke_test"
	ExperimentLandingPage      ExperimentFormat = "landing_page"
	ExperimentCustomerInterview ExperimentFormat = "customer_interview"
	ExperimentWizardOfOz       ExperimentFormat = "wizard_of_oz"
)

// ExperimentBrief is the output of Phase 4b: the cheapest test design
// for resolving assumptions that desk research could not settle.
type ExperimentBrief struct {
	Format        ExperimentFormat `json:"format"`
	Objective     string           `json:"objective"`
	Hypothesis    string           `json:"hypothesis"`
	SuccessMetric string           `json:"success_metric"`
	TimeBoxDays   int              `json:"time_box_days"`
	SetupSteps    []string         `json:"setup_steps"`
	RequiredTools []string         `json:"required_tools"`
	RawClaim      string           `json:"raw_claim"`    // the claim being tested
	CostUSD       float64          `json:"cost_usd"`
}

const experimentSystemPrompt = `You are a lean experiment designer. Given unresolved product assumptions, design the single cheapest experiment that would produce a clear signal within 1–2 weeks.
Respond ONLY with valid JSON — no markdown, no explanation.`

const experimentUserPromptTpl = `Design the cheapest experiment to resolve these unvalidated product assumptions.

PROBLEM: %s
JOBS: %s

UNRESOLVED CLAIMS (insufficient_data):
%s

Choose ONE experiment format:
- "smoke_test": build a no-code landing page or waitlist page to measure demand (best for: demand/pricing signals)
- "landing_page": one-page value proposition site with CTA click measurement (best for: positioning/messaging)
- "customer_interview": 5 structured interviews with target customers (best for: job/pain validation)
- "wizard_of_oz": manually deliver the product behind the scenes to test willingness to pay/use (best for: trust/usage patterns)

Return JSON:
{"format":"smoke_test|landing_page|customer_interview|wizard_of_oz","objective":"one sentence — what will this experiment prove or disprove","hypothesis":"if [we do X], then [Y% of target users] will [measurable action] within [N days]","success_metric":"specific number + metric + time bound","time_box_days":7,"setup_steps":["step 1","step 2","step 3"],"required_tools":["tool 1"],"raw_claim":"the primary assumption being tested"}`

// insufficientClaims returns claims with insufficient_data verdict from a ValidationResult.
func insufficientClaims(v *ValidationResult) []ClaimValidation {
	var out []ClaimValidation
	for _, c := range v.Claims {
		if c.Verdict == VerdictInsufficientData {
			out = append(out, c)
		}
	}
	return out
}

// renderInsufficientClaims formats insufficient_data claims for the experiment prompt.
func renderInsufficientClaims(claims []ClaimValidation) string {
	var b strings.Builder
	for _, c := range claims {
		fmt.Fprintf(&b, "Rank %d: %q (confidence: %.2f)\n  Notes: %s\n\n",
			c.RATRank, c.Claim, c.Confidence, c.Notes)
	}
	return strings.TrimSpace(b.String())
}

// GenerateExperiment designs the cheapest experiment to resolve insufficient_data assumptions.
// It is called only when validation.NeedsExperiment == true.
func GenerateExperiment(ctx context.Context, c *LLMClient, frame *FrameResult, val *ValidationResult) (*ExperimentBrief, error) {
	claims := insufficientClaims(val)
	if len(claims) == 0 {
		return nil, fmt.Errorf("GenerateExperiment called but no insufficient_data claims found")
	}

	jobs := strings.Join(frame.Jobs, "; ")
	rendered := renderInsufficientClaims(claims)

	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultReasonerModel,
		Messages: []Message{
			{Role: "system", Content: experimentSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(experimentUserPromptTpl,
				frame.ProblemStatement, jobs, rendered)},
		},
		MaxTokens:   1000,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("experiment llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var raw ExperimentBrief
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("experiment parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}
	raw.CostUSD = resp.CostUSD
	return &raw, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/discovery/... -run "TestExperimentFormat_Constants" -v
go build ./...
go vet ./...
```
Expected: `TestExperimentFormat_Constants` PASS; integration test skipped (no key).

**Step 5: Commit**

```bash
git add internal/discovery/experiment.go internal/discovery/experiment_test.go
git commit -m "feat: Phase 4b ExperimentBrief types + GenerateExperiment() LLM call"
```

---

### Task 2: Artifacts — Session.Experiment + writeExperiment()

**Files:**
- Modify: `internal/discovery/artifacts.go`
- Modify: `internal/discovery/artifacts_test.go`

**Step 1: Write the failing test**

Add to `internal/discovery/artifacts_test.go`:

```go
func TestArtifacts_WritesExperimentFile(t *testing.T) {
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
		Experiment: &discovery.ExperimentBrief{
			Format:        discovery.ExperimentCustomerInterview,
			Objective:     "validate that founders trust LLM-generated insights",
			Hypothesis:    "if we interview 10 founders, 7 will rate LLM insights as trustworthy",
			SuccessMetric: "7/10 founders rate insights as trustworthy within 7 days",
			TimeBoxDays:   7,
			SetupSteps:    []string{"write interview script", "recruit 10 participants", "run interviews"},
			RequiredTools: []string{"Calendly", "Zoom"},
			RawClaim:      "LLM-generated validation is trusted by founders",
			CostUSD:       0.00080,
		},
	}
	if err := discovery.WriteArtifacts(dir, session); err != nil {
		t.Fatalf("write: %v", err)
	}
	expFile := filepath.Join(dir, "2026-04-08-test-idea-experiment.md")
	if _, err := os.Stat(expFile); err != nil {
		t.Errorf("experiment file not created: %v", err)
	}
	content, _ := os.ReadFile(expFile)
	s := string(content)
	for _, want := range []string{"customer_interview", "validate that founders", "7 days", "Calendly"} {
		if !strings.Contains(s, want) {
			t.Errorf("experiment file missing %q", want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/discovery/... -run TestArtifacts_WritesExperimentFile -v
```
Expected: FAIL — `Session` has no field `Experiment`.

**Step 3: Update artifacts.go**

3a. Add `Experiment *ExperimentBrief` to `Session` struct (after `Validation`):
```go
type Session struct {
	Slug       string
	Date       string
	Frame      *FrameResult
	Hypothesis *HypothesisResult
	Scan       *ScanResult
	Validation *ValidationResult
	Experiment *ExperimentBrief
}
```

3b. In `WriteArtifacts()`, after the Validation block, add:
```go
	if s.Experiment != nil {
		if err := writeExperiment(prefix+"-experiment.md", s.Experiment); err != nil {
			return err
		}
	}
```

3c. Add `writeExperiment()` at the bottom of artifacts.go:
```go
func writeExperiment(path string, e *ExperimentBrief) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Experiment Brief\n\n")
	fmt.Fprintf(&b, "## Format: %s\n\n", e.Format)
	fmt.Fprintf(&b, "**Objective:** %s\n\n", e.Objective)
	fmt.Fprintf(&b, "**Hypothesis:** %s\n\n", e.Hypothesis)
	fmt.Fprintf(&b, "**Success metric:** %s\n\n", e.SuccessMetric)
	fmt.Fprintf(&b, "**Time box:** %d days\n\n", e.TimeBoxDays)
	if e.RawClaim != "" {
		fmt.Fprintf(&b, "**Testing claim:** %s\n\n", e.RawClaim)
	}
	if len(e.SetupSteps) > 0 {
		fmt.Fprintf(&b, "## Setup Steps\n\n")
		for i, s := range e.SetupSteps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, s)
		}
		fmt.Fprintf(&b, "\n")
	}
	if len(e.RequiredTools) > 0 {
		fmt.Fprintf(&b, "## Required Tools\n\n")
		for _, t := range e.RequiredTools {
			fmt.Fprintf(&b, "- %s\n", t)
		}
		fmt.Fprintf(&b, "\n")
	}
	fmt.Fprintf(&b, "---\n\n*Cost: $%.5f*\n", e.CostUSD)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
```

**Step 4: Run all artifact tests**

```bash
go test ./internal/discovery/... -run TestArtifacts -v
go build ./...
go vet ./...
```
Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/discovery/artifacts.go internal/discovery/artifacts_test.go
git commit -m "feat: Session.Experiment field + writeExperiment() artifact"
```

---

### Task 3: CLI — Phase 4b in cmd_discover.go + child beads issue

**Files:**
- Modify: `cmd/sdp/cmd_discover.go`

**Step 1: Read the current file**

Note the current pipeline order. Phase 4b goes AFTER Phase 4a and BEFORE `WriteArtifacts` (so the experiment brief is included in the artifact write).

**Step 2: Insert Phase 4b block**

After the Checkpoint D block (after `fmt.Printf("  Cost:     $%.5f\n\n", validation.CostUSD)`) and before `WriteArtifacts`, insert:

```go
	// ── Phase 4b: EXPERIMENT (conditional on insufficient_data) ──
	if validation.NeedsExperiment {
		fmt.Printf("🧪 Phase 4b: Designing experiment for unresolved assumptions...\n")
		experiment, err := discovery.GenerateExperiment(ctx, client, frame, validation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "   warning: experiment design: %v\n", err)
		} else {
			session.Experiment = experiment
			fmt.Printf("   format:  %s\n", experiment.Format)
			fmt.Printf("   metric:  %s\n", experiment.SuccessMetric)
			fmt.Printf("   timebox: %d days\n", experiment.TimeBoxDays)
			fmt.Printf("   cost:    $%.5f\n\n", experiment.CostUSD)
		}
	}
```

Note: Phase 4b failure is a WARNING (not fatal) — the pipeline continues and writes artifacts without the experiment brief.

**Step 3: Add child beads issue creation for experiment**

After the `WriteArtifacts` call and before the discovery issue creation block, add:

```go
	// ── Create experiment beads issue (if Phase 4b ran) ───────────
	if session.Experiment != nil {
		fmt.Printf("📌 Creating experiment issue...\n")
		expID, err := createExperimentIssue(idea, session.Experiment, absOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "   warning: could not create experiment issue: %v\n", err)
		} else {
			fmt.Printf("   created: %s\n", expID)
		}
	}
```

Add the `createExperimentIssue` function at the bottom of cmd_discover.go:

```go
func createExperimentIssue(idea string, e *discovery.ExperimentBrief, artifactDir string) (string, error) {
	title := fmt.Sprintf("Experiment: %s [%s]", idea, e.Format)
	desc := fmt.Sprintf(
		"## Experiment Brief\n\n**Idea:** %s\n\n**Format:** %s\n\n**Objective:** %s\n\n**Hypothesis:** %s\n\n**Success metric:** %s\n\n**Time box:** %d days\n\n**Testing claim:** %s\n\n**Artifacts:** %s/",
		idea, e.Format, e.Objective, e.Hypothesis, e.SuccessMetric, e.TimeBoxDays, e.RawClaim, artifactDir,
	)
	cmd := exec.Command("bd", "create",
		"--title="+title,
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

**Step 4: Build and vet**

```bash
go build ./...
go vet ./...
```
Expected: clean.

**Step 5: Dogfood smoke test**

```bash
source .env && ./bin/sdp discover "automate product discovery using AI agents" 2>&1 | grep -A5 "Phase 4b\|experiment"
```
Expected: if `needs_experiment=true`, the Phase 4b block runs and logs format + metric + timebox.

**Step 6: Commit**

```bash
git add cmd/sdp/cmd_discover.go
git commit -m "feat: Phase 4b EXPERIMENT in discover pipeline — conditional on insufficient_data"
```

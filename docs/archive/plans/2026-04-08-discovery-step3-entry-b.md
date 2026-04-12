# Discovery Step 3: Entry B — Beads-Driven Auto-Run

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** When a beads issue with `issue_type=discovery` appears in the ready queue, the SDP orchestration loop automatically runs the discovery pipeline (instead of dispatching it to a code agent), then closes the discovery issue and creates a result issue (feature or task depending on verdict).

**Architecture:** Three-file change: (1) `internal/control/control.go` — add `IssueType string` field to `FeatureCard`, (2) `internal/control/repo_beads.go` — map `bd.Type` → `card.IssueType` in `bdToCard`, (3) `internal/executor/loop_v2.go` — after loading card, if `IssueType == "discovery"`, call a new `RunDiscoveryPipeline()` function (in a new file `internal/executor/discovery_runner.go`) instead of the normal clarify/plan/dispatch path.

**Tech Stack:** Go 1.26, `internal/control`, `internal/executor`, `internal/discovery`, `os/exec` (for `bd` CLI), environment variable `OPENROUTER_API_KEY`.

**Important:** `RunDiscoveryPipeline` shells out to the `sdp discover` binary via `exec.Command` to avoid creating a tight coupling between executor and the discovery LLM calls. This is intentional — it is the same pattern already used for `bd create` in the discovery CLI.

---

### Task 1: Add IssueType to FeatureCard + map it

**Files:**
- Modify: `internal/control/control.go`
- Modify: `internal/control/repo_beads.go`

**Step 1: Write the failing test**

Add to `internal/control/repo_beads_test.go` (create if it doesn't exist):

```go
package control_test

import (
	"testing"

	"sdp_dev/internal/control"
)

func TestBdToCard_MapsIssueType(t *testing.T) {
	issue := control.TestBdIssue{
		ID:    "beads-001",
		Title: "Discovery: automate product discovery",
		Type:  "discovery",
	}
	card := control.ExposedBdToCard(issue)
	if card.IssueType != "discovery" {
		t.Errorf("expected IssueType=discovery, got %q", card.IssueType)
	}
}
```

Note: `bdIssue` and `bdToCard` are unexported. If there is no existing test file that tests them, the easiest approach is to add a whitebox test in `package control` (not `package control_test`). Use this instead:

```go
// internal/control/repo_beads_test.go
package control

import (
	"testing"
)

func TestBdToCard_MapsIssueType(t *testing.T) {
	bd := bdIssue{
		ID:    "beads-001",
		Title: "Discovery: automate product discovery",
		Type:  "discovery",
	}
	card := bdToCard(bd)
	if card.IssueType != "discovery" {
		t.Errorf("expected IssueType=discovery, got %q", card.IssueType)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./internal/control/... -run TestBdToCard_MapsIssueType -v
```
Expected: FAIL — `FeatureCard` has no field `IssueType`.

**Step 3: Add IssueType to FeatureCard**

In `internal/control/control.go`, add `IssueType` after `TaskType` (line ~58):
```go
	TaskType                string                 `yaml:"task_type,omitempty" json:"task_type,omitempty"`
	IssueType               string                 `yaml:"issue_type,omitempty" json:"issue_type,omitempty"`
```

**Step 4: Map bd.Type in bdToCard**

In `internal/control/repo_beads.go`, inside `bdToCard()` after `card.ExecutionMode = strconv.Itoa(bd.Priority)`:
```go
	// Map Beads issue_type directly
	if bd.Type != "" {
		card.IssueType = bd.Type
	}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/control/... -run TestBdToCard_MapsIssueType -v
go build ./...
go vet ./...
```
Expected: PASS.

**Step 6: Commit**

```bash
git add internal/control/control.go internal/control/repo_beads.go internal/control/repo_beads_test.go
git commit -m "feat: map Beads issue_type → FeatureCard.IssueType in bdToCard"
```

---

### Task 2: discovery_runner.go — RunDiscoveryFromCard

**Files:**
- Create: `internal/executor/discovery_runner.go`
- Create: `internal/executor/discovery_runner_test.go`

**Context:** The runner shells out to the `sdp` binary (found on PATH or via `os.Executable()` heuristic) rather than calling `internal/discovery` directly. This keeps the executor decoupled from LLM API keys and discovery prompt logic. The runner:
1. Extracts the idea from `card.NormalizedIntent` (the full description). The idea text is in the card title: strip the `"Discovery: "` prefix to get the raw idea.
2. Runs `sdp discover "{idea}"` as a subprocess with inherited env (so `OPENROUTER_API_KEY` is available).
3. Captures exit code: 0 = success, non-zero = failure.
4. On success, updates the card notes with a summary and closes it.

**Step 1: Write the failing test**

```go
// internal/executor/discovery_runner_test.go
package executor

import (
	"testing"

	"sdp_dev/internal/control"
)

func TestExtractDiscoveryIdea_StripPrefix(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Discovery: automate product discovery", "automate product discovery"},
		{"Discovery: build a crypto trading bot", "build a crypto trading bot"},
		{"automate product discovery", "automate product discovery"}, // no prefix — passthrough
		{"", ""},
	}
	for _, tt := range tests {
		got := extractDiscoveryIdea(tt.title)
		if got != tt.expected {
			t.Errorf("extractDiscoveryIdea(%q) = %q, want %q", tt.title, got, tt.expected)
		}
	}
}

func TestIsDiscoveryCard(t *testing.T) {
	yes := &control.FeatureCard{IssueType: "discovery"}
	no := &control.FeatureCard{IssueType: "feature"}
	noType := &control.FeatureCard{}

	if !isDiscoveryCard(yes) {
		t.Error("expected true for IssueType=discovery")
	}
	if isDiscoveryCard(no) {
		t.Error("expected false for IssueType=feature")
	}
	if isDiscoveryCard(noType) {
		t.Error("expected false for empty IssueType")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/executor/... -run "TestExtractDiscoveryIdea|TestIsDiscoveryCard" -v
```
Expected: FAIL — functions undefined.

**Step 3: Write discovery_runner.go**

```go
// internal/executor/discovery_runner.go
package executor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"sdp_dev/internal/control"
)

// isDiscoveryCard returns true if the card is a discovery-type issue.
func isDiscoveryCard(card *control.FeatureCard) bool {
	return card != nil && card.IssueType == "discovery"
}

// extractDiscoveryIdea strips the "Discovery: " prefix from a card title
// to recover the raw idea string passed to `sdp discover`.
func extractDiscoveryIdea(title string) string {
	return strings.TrimPrefix(title, "Discovery: ")
}

// RunDiscoveryFromCard runs the `sdp discover` pipeline for a discovery-typed card.
// It shells out to the sdp binary so that the executor remains decoupled from
// LLM API keys and discovery-specific logic.
//
// On success it closes the beads issue; on failure it logs the error and
// leaves the issue open for manual review.
func RunDiscoveryFromCard(ctx context.Context, store *control.Store, card *control.FeatureCard, projectRoot string) error {
	logger := slog.Default().With("component", "discovery-runner", "card_id", card.ID)

	idea := extractDiscoveryIdea(card.Title)
	if idea == "" {
		return fmt.Errorf("cannot extract idea from card title: %q", card.Title)
	}

	// Find the sdp binary: prefer same directory as the current executable.
	sdpBin, err := findSdpBinary()
	if err != nil {
		return fmt.Errorf("sdp binary not found: %w", err)
	}

	outDir := projectRoot + "/docs/discovery"
	logger.Info("running discovery pipeline", "idea", idea, "out_dir", outDir)

	cmd := exec.CommandContext(ctx, sdpBin, "discover", "--out="+outDir, idea)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ() // inherit OPENROUTER_API_KEY and other env vars

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sdp discover failed: %w", err)
	}

	// Close the discovery issue with a note.
	if store != nil {
		notes := fmt.Sprintf("Discovery pipeline complete. Artifacts at %s/", outDir)
		if saveErr := store.CloseCardWithNotes(card.ID, notes); saveErr != nil {
			logger.Warn("failed to close discovery card", "error", saveErr)
		} else {
			logger.Info("discovery card closed", "card_id", card.ID)
		}
	}
	return nil
}

// findSdpBinary locates the `sdp` binary. It looks in:
//  1. Same directory as the current executable (for `sdp-orchestrate-daemon` co-located with `sdp`)
//  2. PATH
func findSdpBinary() (string, error) {
	// Try same directory as current process
	if self, err := os.Executable(); err == nil {
		dir := self[:strings.LastIndex(self, "/")]
		candidate := dir + "/sdp"
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	// Fall back to PATH
	return exec.LookPath("sdp")
}
```

**Step 4: Add `CloseCardWithNotes` to Store if it doesn't exist**

Check whether `store.CloseCardWithNotes` exists:
```bash
grep -n "CloseCardWithNotes\|CloseCard" /Users/fall_out_bug/projects/vibe_coding/sdp_lab/internal/control/store.go 2>/dev/null | head -10
grep -rn "CloseCardWithNotes\|CloseCard" /Users/fall_out_bug/projects/vibe_coding/sdp_lab/internal/control/ | head -10
```

If it doesn't exist, add a minimal shim. Find the `Store` struct and `SaveCard` method and add:
```go
// CloseCardWithNotes marks a card as done and appends notes.
func (s *Store) CloseCardWithNotes(cardID, notes string) error {
	card, err := s.LoadCard("", cardID)
	if err != nil {
		return fmt.Errorf("load card: %w", err)
	}
	card.Status = "done"
	if notes != "" {
		// Append to existing notes via bd update
		cmd := exec.Command("bd", "update", cardID, "--notes="+notes)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("bd update notes: %w: %s", err, out)
		}
	}
	cmd := exec.Command("bd", "close", cardID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("bd close: %w: %s", err, out)
	}
	_ = card
	return nil
}
```

If `Store` doesn't have exec access directly, use the `BeadsCardRepository` pattern (`runBDWrite`). Look at existing patterns in `internal/control/` and follow them exactly.

**Step 5: Run tests**

```bash
go test ./internal/executor/... -run "TestExtractDiscoveryIdea|TestIsDiscoveryCard" -v
go build ./...
go vet ./...
```
Expected: both PASS.

**Step 6: Commit**

```bash
git add internal/executor/discovery_runner.go internal/executor/discovery_runner_test.go
git commit -m "feat: RunDiscoveryFromCard + isDiscoveryCard + extractDiscoveryIdea helpers"
```

---

### Task 3: Route discovery issues in loop_v2.go

**Files:**
- Modify: `internal/executor/loop_v2.go`

**Step 1: Read loop_v2.go first**

```bash
cat /Users/fall_out_bug/projects/vibe_coding/sdp_lab/internal/executor/loop_v2.go
```

Find the section where the card is loaded after dispatch:
```go
card, loadErr := bridge.Store.LoadCard("", cardID)
if loadErr != nil {
    logger.Error("load card before clarification failed", ...)
    continue
}
// clarify ... plan ... dispatch ...
```

**Step 2: Insert the discovery route immediately after the first LoadCard call**

After the `if loadErr != nil { ... continue }` block for the initial load (not the re-load before planning), add:

```go
			// Route discovery issues to the discovery pipeline instead of normal dispatch.
			if isDiscoveryCard(card) {
				logger.Info("routing discovery issue to pipeline", "card_id", cardID, "idea", card.Title)
				if discErr := RunDiscoveryFromCard(ctx, bridge.Store, card, projectRoot); discErr != nil {
					logger.Error("discovery pipeline failed", "card_id", cardID, "error", discErr)
				}
				continue
			}
```

This `continue` skips the clarify/plan/dispatch flow entirely for discovery issues.

**Step 3: Build and vet**

```bash
go build ./...
go vet ./...
```
Expected: clean.

**Step 4: Manual integration test**

Since this requires a running Dolt server and the full orchestrate daemon, the test is manual:

```bash
# 1. Start the orchestrate daemon in one terminal (if it can start):
# ./bin/sdp-orchestrate-daemon &

# 2. Create a discovery issue:
source .env && bd create --title="Discovery: minimal viable CLI tool" --type=discovery --priority=2

# 3. Watch the daemon logs for "routing discovery issue to pipeline"
```

If Dolt is unreachable (common in local dev), verify the routing logic is correct by reading the code rather than running it end-to-end.

**Step 5: Commit**

```bash
git add internal/executor/loop_v2.go
git commit -m "feat: route IssueType=discovery cards to RunDiscoveryFromCard in loop_v2"
```

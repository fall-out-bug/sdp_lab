# Discovery Step 4: Interactive Checkpoint C (TTY depth decisions)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** When running `sdp discover` in a terminal (TTY), Checkpoint C prompts the user to decide what to do with each flagged scan item (Deep dive / Proceed provisional / Downgrade to MONITOR) instead of silently proceeding with defaults. In non-TTY/agent mode the existing default behaviour is preserved.

**Architecture:** Two components: (1) a `isTerminal()` helper added to `cmd/sdp/` (copies the pattern from `sdp/sdp-plugin/internal/ui/colors.go`), (2) a `resolveCheckpointC(scanResult, isInteractive bool)` function that either prompts stdin for each blocking item or applies automatic defaults. `cmd_discover.go` calls `resolveCheckpointC` instead of just printing `RenderCheckpoint`. No session-state serialisation yet (that is the full async-pause design — deferred).

**Scope (YAGNI):** Interactive stdin prompting for flagged scan items only. No session save/resume. No async pause for headless agents. Those come later.

**Tech Stack:** Go 1.26, `bufio.Scanner` for stdin reading, `os.Stdout.Stat()` for TTY detection, `internal/discovery` types.

---

### Task 1: isTerminal helper + resolveCheckpointC

**Files:**
- Create: `cmd/sdp/terminal.go`
- Create: `cmd/sdp/checkpoint_c.go`
- Create: `cmd/sdp/checkpoint_c_test.go`

**Step 1: Write the failing tests**

```go
// cmd/sdp/checkpoint_c_test.go
package main

import (
	"strings"
	"testing"

	"sdp_dev/internal/discovery"
)

// makeTestScanResult builds a ScanResult with one flagged + one settled item.
func makeTestScanResult() *discovery.ScanResult {
	flag := &discovery.DepthFlag{
		Flagged:  true,
		Blocking: true,
		Reason:   "no_primary_source",
	}
	return &discovery.ScanResult{
		Items: []discovery.ScanItem{
			{Name: "SettledTool", Disposition: discovery.DispositionInspire, CoverageScore: 0.7},
			{Name: "FlaggedTool", Disposition: discovery.DispositionAdopt, CoverageScore: 0.04, DepthFlag: flag},
		},
		Whitespace: "gap description",
	}
}

func TestResolveCheckpointC_NonInteractiveUsesDefaults(t *testing.T) {
	scan := makeTestScanResult()
	// Non-interactive: should apply default resolution (proceed provisional) without blocking.
	resolutions := resolveCheckpointC(scan, false, nil)
	// FlaggedTool should get a resolution.
	if _, ok := resolutions["FlaggedTool"]; !ok {
		t.Error("expected resolution for FlaggedTool in non-interactive mode")
	}
	// SettledTool should have no resolution (it is not flagged).
	if _, ok := resolutions["SettledTool"]; ok {
		t.Error("unexpected resolution for SettledTool (not flagged)")
	}
}

func TestResolveCheckpointC_DefaultResolutionIsProceedProvisional(t *testing.T) {
	scan := makeTestScanResult()
	resolutions := resolveCheckpointC(scan, false, nil)
	res := resolutions["FlaggedTool"]
	if !strings.Contains(res, "proceed") && !strings.Contains(res, "provisional") {
		t.Errorf("expected proceed_provisional default, got %q", res)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_lab
go test ./cmd/sdp/... -run "TestResolveCheckpointC" -v
```
Expected: FAIL — `resolveCheckpointC` undefined.

**Step 3: Write terminal.go**

```go
// cmd/sdp/terminal.go
package main

import "os"

// isTerminal returns true if stdout is connected to an interactive terminal.
// Copies the pattern from sdp/sdp-plugin/internal/ui/colors.go.
func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
```

**Step 4: Write checkpoint_c.go**

```go
// cmd/sdp/checkpoint_c.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"sdp_dev/internal/discovery"
)

const (
	resolutionProceedProvisional = "proceed_provisional"
	resolutionDeepDive           = "deep_dive"
	resolutionDowngrade          = "downgrade"
)

// resolveCheckpointC decides how to handle each flagged scan item.
//
// In interactive mode (isInteractive=true), it prompts the user at stdin for
// each flagged item with options [D]eep dive / [P]roceed provisional / [I]gnore.
// In non-interactive mode (isInteractive=false), it silently applies
// "proceed_provisional" for all flagged items.
//
// The reader parameter allows test injection of stdin. Pass nil to use os.Stdin.
//
// Returns a map of item.Name → resolution string.
func resolveCheckpointC(scan *discovery.ScanResult, isInteractive bool, reader io.Reader) map[string]string {
	resolutions := make(map[string]string)

	flagged := scan.Flagged()
	if len(flagged) == 0 {
		return resolutions
	}

	if !isInteractive {
		for _, item := range flagged {
			resolutions[item.Name] = resolutionProceedProvisional
		}
		return resolutions
	}

	// Interactive: prompt for each flagged item.
	if reader == nil {
		reader = os.Stdin
	}
	scanner := bufio.NewScanner(reader)

	for _, item := range flagged {
		blocking := ""
		if item.DepthFlag != nil && item.DepthFlag.Blocking {
			blocking = " ⚠️  BLOCKING"
		}
		fmt.Printf("\n  %s%s\n", item.Name, blocking)
		if item.DepthFlag != nil {
			fmt.Printf("  reason: %s\n", item.DepthFlag.Reason)
		}
		fmt.Printf("  [D] Deep dive now  [P] Proceed provisional  [I] Downgrade to MONITOR\n")
		fmt.Printf("  Choice (D/P/I, default=P): ")

		choice := "P"
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				choice = strings.ToUpper(line[:1])
			}
		}

		switch choice {
		case "D":
			resolutions[item.Name] = resolutionDeepDive
		case "I":
			resolutions[item.Name] = resolutionDowngrade
		default:
			resolutions[item.Name] = resolutionProceedProvisional
		}
	}
	return resolutions
}

// printResolutionSummary prints what was decided for each flagged item.
func printResolutionSummary(resolutions map[string]string) {
	if len(resolutions) == 0 {
		return
	}
	fmt.Printf("\n  Depth resolutions applied:\n")
	for name, res := range resolutions {
		icon := "→"
		switch res {
		case resolutionDeepDive:
			icon = "🔍"
		case resolutionDowngrade:
			icon = "↓"
		case resolutionProceedProvisional:
			icon = "⚡"
		}
		fmt.Printf("    %s %s: %s\n", icon, name, res)
	}
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./cmd/sdp/... -run "TestResolveCheckpointC" -v
go build ./...
go vet ./...
```
Expected: both PASS.

**Step 6: Commit**

```bash
git add cmd/sdp/terminal.go cmd/sdp/checkpoint_c.go cmd/sdp/checkpoint_c_test.go
git commit -m "feat: isTerminal() helper + resolveCheckpointC() for interactive depth decisions"
```

---

### Task 2: Wire resolveCheckpointC into cmd_discover.go

**Files:**
- Modify: `cmd/sdp/cmd_discover.go`

**Step 1: Read cmd_discover.go**

Find the Checkpoint C block:
```go
fmt.Println(discovery.RenderCheckpoint(scanResult))
```

**Step 2: Replace the Checkpoint C print with the interactive resolver**

Replace the single `fmt.Println(discovery.RenderCheckpoint(scanResult))` line with:

```go
	// ── Checkpoint C: Depth decisions ─────────────────────────────
	fmt.Println(discovery.RenderCheckpoint(scanResult))
	interactive := isTerminal()
	if !interactive {
		fmt.Printf("   (non-interactive mode — proceeding with defaults for all flagged items)\n\n")
	}
	resolutions := resolveCheckpointC(scanResult, interactive, nil)
	if interactive && len(resolutions) > 0 {
		printResolutionSummary(resolutions)
		fmt.Println()
	} else if !interactive && len(resolutions) > 0 {
		printResolutionSummary(resolutions)
	}
```

Note: `RenderCheckpoint` still runs (it prints the full two-section view). The interactive prompting happens AFTER the render, for each flagged item.

**Step 3: Build and vet**

```bash
go build ./...
go vet ./...
```
Expected: clean.

**Step 4: Manual smoke test in TTY**

```bash
source .env && ./bin/sdp discover "build a personal finance tracker" 2>&1
```

At Checkpoint C, when flagged items appear, you should see:
```
  FlaggedTool ⚠️  BLOCKING
  reason: no_primary_source
  [D] Deep dive now  [P] Proceed provisional  [I] Downgrade to MONITOR
  Choice (D/P/I, default=P): 
```
Type `P` and press Enter for each. Pipeline should continue.

If running non-interactively (piped output), the block prints defaults and continues without waiting.

**Step 5: Commit**

```bash
git add cmd/sdp/cmd_discover.go
git commit -m "feat: interactive Checkpoint C depth decisions with TTY detection"
```

---

### Task 3: Handle deep_dive + downgrade resolutions

**Files:**
- Modify: `cmd/sdp/cmd_discover.go` (add post-resolution handling)
- Modify: `internal/discovery/scan.go` (add `ApplyResolution` helper)

**Context:** Currently `resolveCheckpointC` records decisions but does not act on them. For `downgrade`, we need to change the item's disposition to `MONITOR`. For `deep_dive`, the ideal is a real-time web lookup — that is Phase 3 depth research and is out of scope here. For now, `deep_dive` is treated as `proceed_provisional` (log a TODO note).

**Step 1: Write the failing test**

Add to `internal/discovery/scan_test.go` (or create `internal/discovery/scan_resolution_test.go`):

```go
func TestApplyResolutions_DowngradeChangesDisposition(t *testing.T) {
	item := discovery.ScanItem{
		Name:        "FlaggedTool",
		Disposition: discovery.DispositionAdopt,
		DepthFlag:   &discovery.DepthFlag{Flagged: true, Reason: "no_primary_source"},
	}
	result := &discovery.ScanResult{Items: []discovery.ScanItem{item}}

	resolutions := map[string]string{
		"FlaggedTool": "downgrade",
	}
	updated := discovery.ApplyResolutions(result, resolutions)
	if updated.Items[0].Disposition != discovery.DispositionMonitor {
		t.Errorf("expected MONITOR after downgrade, got %s", updated.Items[0].Disposition)
	}
}

func TestApplyResolutions_ProceedProvisionalPreservesDisposition(t *testing.T) {
	item := discovery.ScanItem{
		Name:        "FlaggedTool",
		Disposition: discovery.DispositionAdopt,
		DepthFlag:   &discovery.DepthFlag{Flagged: true},
	}
	result := &discovery.ScanResult{Items: []discovery.ScanItem{item}}
	resolutions := map[string]string{"FlaggedTool": "proceed_provisional"}
	updated := discovery.ApplyResolutions(result, resolutions)
	if updated.Items[0].Disposition != discovery.DispositionAdopt {
		t.Errorf("expected ADOPT preserved, got %s", updated.Items[0].Disposition)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/discovery/... -run "TestApplyResolutions" -v
```
Expected: FAIL — `ApplyResolutions` undefined.

**Step 3: Add ApplyResolutions to internal/discovery/scan.go**

```go
// ApplyResolutions applies human checkpoint-C decisions to a ScanResult.
// "downgrade" → change disposition to MONITOR and unflag the item.
// "proceed_provisional" → leave disposition unchanged, keep DepthFlag.
// "deep_dive" → treated as proceed_provisional for now (real deep dive is future work).
func ApplyResolutions(r *ScanResult, resolutions map[string]string) *ScanResult {
	if len(resolutions) == 0 {
		return r
	}
	updated := &ScanResult{
		Items:            make([]ScanItem, len(r.Items)),
		Whitespace:       r.Whitespace,
		RecommendedStack: r.RecommendedStack,
		CostUSD:          r.CostUSD,
	}
	copy(updated.Items, r.Items)
	for i, item := range updated.Items {
		res, ok := resolutions[item.Name]
		if !ok {
			continue
		}
		switch res {
		case "downgrade":
			updated.Items[i].Disposition = DispositionMonitor
			updated.Items[i].DepthFlag = nil // clear flag — decision made
		case "deep_dive":
			// Treated as proceed_provisional until deep dive is implemented.
			// Flag is preserved; user can re-run with real sources.
		}
		// proceed_provisional: no change to disposition or flag
	}
	return updated
}
```

**Step 4: Wire ApplyResolutions in cmd_discover.go**

After `printResolutionSummary(resolutions)` in the Checkpoint C block, add:
```go
	if len(resolutions) > 0 {
		scanResult = discovery.ApplyResolutions(scanResult, resolutions)
		session.Scan = scanResult
	}
```

**Step 5: Run all tests**

```bash
go test ./internal/discovery/... -run "TestApplyResolutions" -v
go test ./cmd/sdp/... -run "TestResolveCheckpointC" -v
go build ./...
go vet ./...
```
Expected: all PASS.

**Step 6: Commit**

```bash
git add internal/discovery/scan.go cmd/sdp/cmd_discover.go
git commit -m "feat: ApplyResolutions() — downgrade/proceed_provisional depth decisions affect ScanResult"
```

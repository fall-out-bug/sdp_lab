# Discovery Step 1: GO Verdict → Feature Beads Issue

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** When the discovery pipeline returns a GO verdict, create a `type=feature` beads issue (instead of `type=task`) that includes the functional requirements from the hypothesis so the SDP orchestration loop can pick it up as a real feature.

**Architecture:** One function change in `cmd/sdp/cmd_discover.go`. `createDiscoveryIssue` grows a `verdict` parameter; on GO it emits `--type=feature` with requirements formatted as a bullet list, on PIVOT/KILL it falls back to `--type=task` with the original summary. No new files.

**Tech Stack:** Go 1.26, `os/exec`, `internal/discovery` types.

---

### Task 1: Wire the verdict into createDiscoveryIssue

**Files:**
- Modify: `cmd/sdp/cmd_discover.go`

**Step 1: Read the current file**

```bash
cat cmd/sdp/cmd_discover.go
```

Note the current signature of `createDiscoveryIssue` (line ~144) and the call site in `runDiscover` (line ~136).

**Step 2: Write a unit test first**

There is no existing unit test for `createDiscoveryIssue` because it shells out to `bd`. We cannot unit-test the shell-out itself without a mock, so we validate the *description string building* instead. Add to a new file `cmd/sdp/cmd_discover_test.go`:

```go
package main

import (
	"strings"
	"testing"

	"sdp_dev/internal/discovery"
)

func TestBuildFeatureDescription_GOIncludesRequirements(t *testing.T) {
	frame := &discovery.FrameResult{
		ProblemStatement: "developers waste time on manual discovery",
		Appetite:         "medium",
	}
	hyp := &discovery.HypothesisResult{
		WeBelieve: "solo founders need faster validation",
		Requirements: []string{
			"user can upload transcript and receive summary",
			"CLI produces markdown artifact in under 60s",
		},
	}
	val := &discovery.ValidationResult{
		FinalVerdict:  discovery.VerdictGO,
		VerdictReason: "both core assumptions supported",
	}
	desc := buildDiscoveryDescription("automate product discovery", frame, hyp, val, "/tmp/out")

	if !strings.Contains(desc, "GO") {
		t.Error("description missing GO verdict")
	}
	if !strings.Contains(desc, "user can upload transcript") {
		t.Error("description missing requirements")
	}
	if !strings.Contains(desc, "CLI produces markdown artifact") {
		t.Error("description missing second requirement")
	}
}

func TestBuildFeatureDescription_PIVOTOmitsRequirements(t *testing.T) {
	frame := &discovery.FrameResult{
		ProblemStatement: "developers waste time",
		Appetite:         "small",
	}
	hyp := &discovery.HypothesisResult{
		WeBelieve:    "founders need help",
		Requirements: []string{"some requirement"},
	}
	val := &discovery.ValidationResult{
		FinalVerdict:    discovery.VerdictPIVOT,
		VerdictReason:   "evidence mixed",
		PivotSuggestion: "narrow to research repo",
	}
	desc := buildDiscoveryDescription("some idea", frame, hyp, val, "/tmp/out")

	if !strings.Contains(desc, "PIVOT") {
		t.Error("description missing PIVOT verdict")
	}
	if !strings.Contains(desc, "narrow to research repo") {
		t.Error("description missing pivot suggestion")
	}
}
```

**Step 3: Run test to verify it fails**

```bash
go test ./cmd/sdp/... -run "TestBuildFeatureDescription" -v
```
Expected: FAIL — `buildDiscoveryDescription` undefined.

**Step 4: Implement the changes**

In `cmd/sdp/cmd_discover.go`, make three changes:

4a. Extract a pure `buildDiscoveryDescription` function (testable without exec):

```go
// buildDiscoveryDescription constructs the beads issue description for a discovery run.
// On GO verdict, includes hypothesis requirements for the feature backlog.
func buildDiscoveryDescription(
	idea string,
	frame *discovery.FrameResult,
	hyp *discovery.HypothesisResult,
	val *discovery.ValidationResult,
	artifactDir string,
) string {
	verdictSection := ""
	if val != nil {
		verdictSection = fmt.Sprintf("\n\n**Verdict:** %s — %s", val.FinalVerdict, val.VerdictReason)
		if val.PivotSuggestion != "" {
			verdictSection += fmt.Sprintf("\n\n**Pivot:** %s", val.PivotSuggestion)
		}
		if val.KillReason != "" {
			verdictSection += fmt.Sprintf("\n\n**Kill reason:** %s", val.KillReason)
		}
	}

	hypoSection := ""
	if hyp != nil && hyp.WeBelieve != "" {
		riskiest := "—"
		if len(hyp.Assumptions) > 0 {
			riskiest = hyp.Assumptions[0].Statement
		}
		hypoSection = fmt.Sprintf("\n\n**Hypothesis:** %s\n\n**Riskiest assumption:** %s",
			hyp.WeBelieve, riskiest)
	}

	reqSection := ""
	if val != nil && val.FinalVerdict == discovery.VerdictGO && hyp != nil && len(hyp.Requirements) > 0 {
		var sb strings.Builder
		sb.WriteString("\n\n## Requirements\n\n")
		for _, r := range hyp.Requirements {
			fmt.Fprintf(&sb, "- %s\n", r)
		}
		reqSection = sb.String()
	}

	return fmt.Sprintf(
		"## Discovery: %s\n\n**Problem:** %s\n\n**Appetite:** %s%s%s%s\n\n**Artifacts:** %s/",
		idea, frame.ProblemStatement, frame.Appetite,
		hypoSection, verdictSection, reqSection, artifactDir,
	)
}
```

4b. Rewrite `createDiscoveryIssue` to use the extracted function and choose issue type based on verdict:

```go
func createDiscoveryIssue(
	idea string,
	frame *discovery.FrameResult,
	hypothesis *discovery.HypothesisResult,
	validation *discovery.ValidationResult,
	artifactDir string,
) (string, error) {
	issueType := "task"
	title := "Discovery: " + idea
	if validation != nil && validation.FinalVerdict == discovery.VerdictGO {
		issueType = "feature"
		title = "Feature: " + idea
	}

	desc := buildDiscoveryDescription(idea, frame, hypothesis, validation, artifactDir)

	cmd := exec.Command("bd", "create",
		"--title="+title,
		"--description="+desc,
		"--type="+issueType,
		"--priority=2",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bd create: %w: %s", err, out)
	}
	return string(out), nil
}
```

4c. Update the call site in `runDiscover` to pass `validation`:

Find the existing call (currently ~line 136):
```go
issueID, err := createDiscoveryIssue(idea, frame, hypothesis, absOut)
```
Change to:
```go
issueID, err := createDiscoveryIssue(idea, frame, hypothesis, validation, absOut)
```

**Step 5: Run tests to verify they pass**

```bash
go test ./cmd/sdp/... -run "TestBuildFeatureDescription" -v
go build ./...
go vet ./...
```
Expected: both tests PASS, build and vet clean.

**Step 6: Quick smoke test**

```bash
source .env && ./bin/sdp discover "automate product discovery using AI agents" 2>&1 | tail -15
```
Expected output ends with:
- `📌 Creating beads issue...`  
- `   created: beads-XXX` (or warning if Dolt unreachable)
- If Dolt unreachable, verify the correct `bd create --type=feature --title="Feature: ..."` command would have been called via print debugging if needed.

**Step 7: Commit**

```bash
git add cmd/sdp/cmd_discover.go cmd/sdp/cmd_discover_test.go
git commit -m "feat: GO verdict creates type=feature beads issue with requirements"
```

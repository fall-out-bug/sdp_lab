# Discovery Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a working `sdp discover "idea"` command that runs Phase 1 (FRAME) → Phase 3 (SCAN with depth signals) → checkpoint output, writing artifacts to `docs/discovery/` and a beads issue.

**Architecture:** New `internal/discovery/` package with three layers: `llm.go` (OpenRouter calls via net/http), `scan.go` (parallel MCP fan-out + GPT Researcher context), `depth.go` (coverage envelope + 7 heuristics). CLI entry at `cmd/sdp/cmd_discover.go`. No new dependencies — net/http only.

**Tech Stack:** Go 1.26, OpenRouter API (direct HTTP), `bd` CLI (shell-out for beads), existing `internal/sdputil` for JSON/file ops.

---

## Task 1: Package skeleton + LLM client

**Files:**
- Create: `internal/discovery/discovery.go`
- Create: `internal/discovery/llm.go`
- Create: `internal/discovery/llm_test.go`

**Step 1: Write the failing test**

```go
// internal/discovery/llm_test.go
package discovery_test

import (
    "context"
    "os"
    "testing"
    "github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestLLMClientChat_ReturnsJSON(t *testing.T) {
    key := os.Getenv("OPENROUTER_API_KEY")
    if key == "" {
        t.Skip("OPENROUTER_API_KEY not set")
    }
    c := discovery.NewLLMClient(key, "https://openrouter.ai/api/v1")
    resp, err := c.Chat(context.Background(), discovery.ChatRequest{
        Model: "deepseek/deepseek-v3.2",
        Messages: []discovery.Message{
            {Role: "system", Content: "Reply with valid JSON only."},
            {Role: "user", Content: `Return {"ok":true}`},
        },
        MaxTokens:   100,
        Temperature: 0.0,
    })
    if err != nil {
        t.Fatalf("chat: %v", err)
    }
    if resp.Content == "" {
        t.Fatal("empty content")
    }
    if resp.CostUSD < 0 {
        t.Fatal("negative cost")
    }
}
```

**Step 2: Run to confirm it fails**

```bash
go test ./internal/discovery/... -run TestLLMClientChat -v
```
Expected: `package discovery_test: cannot find package`

**Step 3: Create package skeleton and LLM client**

```go
// internal/discovery/discovery.go
package discovery

const DefaultPlannerModel    = "deepseek/deepseek-v3.2"
const DefaultSynthModel      = "deepseek/deepseek-v3.2"
const DefaultReasonerModel   = "openai/gpt-5.4-mini"
const DefaultOpenRouterBase  = "https://openrouter.ai/api/v1"
```

```go
// internal/discovery/llm.go
package discovery

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    MaxTokens   int       `json:"max_tokens"`
    Temperature float64   `json:"temperature"`
}

type ChatResponse struct {
    Content      string
    InputTokens  int
    OutputTokens int
    CostUSD      float64
    FinishReason string
}

type LLMClient struct {
    apiKey  string
    baseURL string
    http    *http.Client
}

func NewLLMClient(apiKey, baseURL string) *LLMClient {
    return &LLMClient{
        apiKey:  apiKey,
        baseURL: baseURL,
        http:    &http.Client{Timeout: 120 * time.Second},
    }
}

func (c *LLMClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal: %w", err)
    }
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
        c.baseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("new request: %w", err)
    }
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("HTTP-Referer", "https://github.com/sdp-lab")
    httpReq.Header.Set("X-Title", "SDP Discovery")

    resp, err := c.http.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("http: %w", err)
    }
    defer resp.Body.Close()
    raw, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("status %d: %s", resp.StatusCode, raw)
    }

    var out struct {
        Choices []struct {
            Message      Message `json:"message"`
            FinishReason string  `json:"finish_reason"`
        } `json:"choices"`
        Usage struct {
            PromptTokens     int     `json:"prompt_tokens"`
            CompletionTokens int     `json:"completion_tokens"`
            Cost             float64 `json:"cost"`
        } `json:"usage"`
    }
    if err := json.Unmarshal(raw, &out); err != nil {
        return nil, fmt.Errorf("unmarshal: %w", err)
    }
    if len(out.Choices) == 0 {
        return nil, fmt.Errorf("no choices in response")
    }
    return &ChatResponse{
        Content:      out.Choices[0].Message.Content,
        FinishReason: out.Choices[0].FinishReason,
        InputTokens:  out.Usage.PromptTokens,
        OutputTokens: out.Usage.CompletionTokens,
        CostUSD:      out.Usage.Cost,
    }, nil
}
```

**Step 4: Run test**

```bash
source .env && go test ./internal/discovery/... -run TestLLMClientChat -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/discovery/
git commit -m "feat(discovery): LLM client for OpenRouter"
```

---

## Task 2: Phase 1 FRAME

**Files:**
- Create: `internal/discovery/frame.go`
- Create: `internal/discovery/frame_test.go`

**Step 1: Write the failing test**

```go
// internal/discovery/frame_test.go
package discovery_test

import (
    "context"
    "os"
    "testing"
    "github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestFrame_ProducesValidOutput(t *testing.T) {
    key := os.Getenv("OPENROUTER_API_KEY")
    if key == "" {
        t.Skip("OPENROUTER_API_KEY not set")
    }
    c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
    result, err := discovery.Frame(context.Background(), c, "automate code review using AI agents")
    if err != nil {
        t.Fatalf("frame: %v", err)
    }
    if result.ProblemStatement == "" {
        t.Error("empty problem_statement")
    }
    if len(result.Jobs) == 0 {
        t.Error("no jobs identified")
    }
    if result.Appetite == "" {
        t.Error("empty appetite")
    }
    t.Logf("frame result: %+v", result)
}
```

**Step 2: Run to confirm it fails**

```bash
go test ./internal/discovery/... -run TestFrame -v
```
Expected: FAIL — `Frame` undefined

**Step 3: Implement**

```go
// internal/discovery/frame.go
package discovery

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
)

type FrameResult struct {
    ProblemStatement string   `json:"problem_statement"`
    Jobs             []string `json:"jobs"`        // JTBD: who does what, to achieve what
    Appetite         string   `json:"appetite"`    // small/medium/large
    Scope            string   `json:"scope"`       // what's in, what's out
    RawIdea          string   `json:"raw_idea"`
}

const frameSystemPrompt = `You are a product discovery agent specializing in problem framing.
Respond ONLY with valid JSON — no markdown, no explanation.`

const frameUserPromptTpl = `Frame this raw idea into a structured problem.

RAW IDEA: %s

Return JSON:
{"problem_statement":"string","jobs":["who does what to achieve what"],"appetite":"small|medium|large","scope":"string"}`

func Frame(ctx context.Context, c *LLMClient, idea string) (*FrameResult, error) {
    resp, err := c.Chat(ctx, ChatRequest{
        Model:       DefaultPlannerModel,
        Messages: []Message{
            {Role: "system", Content: frameSystemPrompt},
            {Role: "user", Content: fmt.Sprintf(frameUserPromptTpl, idea)},
        },
        MaxTokens:   800,
        Temperature: 0.1,
    })
    if err != nil {
        return nil, fmt.Errorf("frame llm: %w", err)
    }
    content := strings.TrimSpace(resp.Content)
    var result FrameResult
    if err := json.Unmarshal([]byte(content), &result); err != nil {
        return nil, fmt.Errorf("frame parse (finish=%s): %w\ncontent: %s",
            resp.FinishReason, err, content)
    }
    result.RawIdea = idea
    return &result, nil
}
```

**Step 4: Run test**

```bash
source .env && go test ./internal/discovery/... -run TestFrame -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/discovery/frame.go internal/discovery/frame_test.go
git commit -m "feat(discovery): Phase 1 FRAME — problem framing via LLM"
```

---

## Task 3: Depth signal types + 7 heuristics

**Files:**
- Create: `internal/discovery/depth.go`
- Create: `internal/discovery/depth_test.go`

**Step 1: Write the failing tests**

```go
// internal/discovery/depth_test.go
package discovery_test

import (
    "testing"
    "github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestDepthFlag_H3_NoVerdict_Without_PrimarySource(t *testing.T) {
    item := discovery.ScanItem{
        Name:              "DeerFlow",
        Disposition:       discovery.DispositionExtract,
        Stars:             50000,
        PrimarySourceRead: false,
        DescSentences:     3,
        SourceCount:       4,
    }
    flag := discovery.EvalDepth(item)
    if !flag.Flagged {
        t.Error("H3: EXTRACT verdict without primary source must be flagged")
    }
    if flag.Reason != "no_primary_source" {
        t.Errorf("expected reason no_primary_source, got %s", flag.Reason)
    }
}

func TestDepthFlag_H1_HighStarsLowDescription(t *testing.T) {
    item := discovery.ScanItem{
        Name:              "SomeTool",
        Disposition:       discovery.DispositionMonitor,
        Stars:             10000,
        PrimarySourceRead: true,
        DescSentences:     2, // < 5
        SourceCount:       2,
    }
    flag := discovery.EvalDepth(item)
    if !flag.Flagged {
        t.Error("H1: >5K stars + <5 desc sentences must be flagged")
    }
}

func TestDepthFlag_Settled_NoFlag(t *testing.T) {
    item := discovery.ScanItem{
        Name:               "WellResearched",
        Disposition:        discovery.DispositionInspire,
        Stars:              1000,
        PrimarySourceRead:  true,
        ArchitectureReviewed: true,
        DescSentences:      15,
        SourceCount:        3,
        MultiSource:        true,
    }
    flag := discovery.EvalDepth(item)
    if flag.Flagged {
        t.Errorf("well-researched item should not be flagged, got: %s", flag.Reason)
    }
}

func TestCoverageScore(t *testing.T) {
    item := discovery.ScanItem{
        PrimarySourceRead:    true,
        ArchitectureReviewed: true,
        DescSentences:        20,
        MultiSource:          true,
    }
    score := discovery.CoverageScore(item)
    if score < 0.9 {
        t.Errorf("fully covered item should score ≥0.9, got %.2f", score)
    }
}
```

**Step 2: Run to confirm fail**

```bash
go test ./internal/discovery/... -run TestDepth -v
go test ./internal/discovery/... -run TestCoverage -v
```
Expected: FAIL — types undefined

**Step 3: Implement**

```go
// internal/discovery/depth.go
package discovery

type Disposition string

const (
    DispositionAdopt   Disposition = "ADOPT"
    DispositionExtract Disposition = "EXTRACT"
    DispositionInspire Disposition = "INSPIRE"
    DispositionMonitor Disposition = "MONITOR"
    DispositionIgnore  Disposition = "IGNORE"
)

// ScanItem represents one candidate from Phase 3 scan.
type ScanItem struct {
    Name                 string      `json:"name"`
    Disposition          Disposition `json:"disposition"`
    DispositionConfidence float64    `json:"disposition_confidence"`
    Stars                int         `json:"stars"`
    SourceCount          int         `json:"source_count"`
    PrimarySourceRead    bool        `json:"primary_source_read"`
    ArchitectureReviewed bool        `json:"architecture_reviewed"`
    DescSentences        int         `json:"desc_sentences"`
    MultiSource          bool        `json:"multi_source"`
    AgeMonths            int         `json:"age_months"` // 0 = unknown
    // populated after eval
    CoverageScore float64    `json:"coverage_score"`
    DepthFlag     *DepthFlag `json:"depth_flag,omitempty"`
    // output fields
    KeyStrength  string   `json:"key_strength"`
    KeyGap       string   `json:"key_gap"`
    CoversPhases []string `json:"covers_phases"`
}

type DepthFlag struct {
    Flagged           bool   `json:"flagged"`
    Reason            string `json:"reason"`
    RecommendedAction string `json:"recommended_action"` // deep_dive|proceed_provisional|downgrade
    Blocking          bool   `json:"blocking"`
}

// CoverageScore returns 0.0–1.0. Four equally weighted components.
func CoverageScore(item ScanItem) float64 {
    score := 0.0
    if item.PrimarySourceRead {
        score += 0.25
    }
    if item.ArchitectureReviewed {
        score += 0.25
    }
    if item.MultiSource {
        score += 0.25
    }
    // desc length: 0–20+ sentences → 0–0.25
    sentences := item.DescSentences
    if sentences > 20 {
        sentences = 20
    }
    score += float64(sentences) / 20.0 * 0.25
    return score
}

// EvalDepth applies the 7 heuristics and returns a DepthFlag.
func EvalDepth(item ScanItem) DepthFlag {
    cs := CoverageScore(item)

    // H3: universal stop — non-IGNORE verdict without primary source read
    if item.Disposition != DispositionIgnore && !item.PrimarySourceRead {
        return DepthFlag{
            Flagged:           true,
            Reason:            "no_primary_source",
            RecommendedAction: "deep_dive",
            Blocking:          item.Disposition == DispositionAdopt || item.Disposition == DispositionExtract,
        }
    }

    // H4: ADOPT/EXTRACT requires architecture review
    if (item.Disposition == DispositionAdopt || item.Disposition == DispositionExtract) &&
        !item.ArchitectureReviewed {
        return DepthFlag{
            Flagged:           true,
            Reason:            "architecture_not_reviewed",
            RecommendedAction: "deep_dive",
            Blocking:          true,
        }
    }

    // H7: low confidence on high-stakes verdict
    if (item.Disposition == DispositionAdopt || item.Disposition == DispositionExtract) &&
        item.DispositionConfidence > 0 && item.DispositionConfidence < 0.5 {
        return DepthFlag{
            Flagged:           true,
            Reason:            "low_confidence_high_stakes",
            RecommendedAction: "deep_dive",
        }
    }

    // H1: high stars, thin description
    if item.Stars > 5000 && item.DescSentences < 5 {
        return DepthFlag{
            Flagged:           true,
            Reason:            "high_stars_low_description",
            RecommendedAction: "deep_dive",
        }
    }

    // H2: multi-source mention, none read directly
    if item.SourceCount >= 3 && !item.PrimarySourceRead {
        return DepthFlag{
            Flagged:           true,
            Reason:            "multi_source_no_primary",
            RecommendedAction: "deep_dive",
        }
    }

    // H5: recently released, sparse data
    if item.AgeMonths > 0 && item.AgeMonths <= 6 && item.DescSentences < 10 {
        return DepthFlag{
            Flagged:           true,
            Reason:            "recent_sparse",
            RecommendedAction: "deep_dive",
        }
    }

    // H6: low coverage on settled items
    if cs < 0.4 && item.Disposition != DispositionIgnore && item.Disposition != DispositionMonitor {
        return DepthFlag{
            Flagged:           true,
            Reason:            "low_coverage_score",
            RecommendedAction: "proceed_provisional",
        }
    }

    return DepthFlag{Flagged: false}
}
```

**Step 4: Run tests**

```bash
go test ./internal/discovery/... -run "TestDepth|TestCoverage" -v
```
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/discovery/depth.go internal/discovery/depth_test.go
git commit -m "feat(discovery): depth signal mechanism — coverage score + 7 heuristics"
```

---

## Task 4: Phase 3 SCAN — LLM-based scan synthesis

**Files:**
- Create: `internal/discovery/scan.go`
- Create: `internal/discovery/scan_test.go`

**Step 1: Write the failing test**

```go
// internal/discovery/scan_test.go
package discovery_test

import (
    "context"
    "os"
    "testing"
    "github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestScan_ProducesItemsWithCoverage(t *testing.T) {
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
    result, err := discovery.Scan(context.Background(), c, frame)
    if err != nil {
        t.Fatalf("scan: %v", err)
    }
    if len(result.Items) == 0 {
        t.Fatal("scan returned no items")
    }
    // verify depth evaluation ran
    for _, item := range result.Items {
        if item.CoverageScore == 0 && item.Disposition != discovery.DispositionIgnore {
            t.Errorf("item %s: coverage score not set", item.Name)
        }
    }
    settled := result.Settled()
    flagged := result.Flagged()
    t.Logf("settled=%d flagged=%d whitespace=%s", len(settled), len(flagged), result.Whitespace)
}
```

**Step 2: Run to confirm fail**

```bash
go test ./internal/discovery/... -run TestScan -v
```
Expected: FAIL — `Scan` undefined

**Step 3: Implement**

```go
// internal/discovery/scan.go
package discovery

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
)

type ScanResult struct {
    Items      []ScanItem `json:"items"`
    Whitespace string     `json:"whitespace"`
    RecommendedStack []string `json:"recommended_stack"`
    CostUSD    float64    `json:"cost_usd"`
}

func (r *ScanResult) Settled() []ScanItem {
    var out []ScanItem
    for _, item := range r.Items {
        if item.DepthFlag == nil || !item.DepthFlag.Flagged {
            out = append(out, item)
        }
    }
    return out
}

func (r *ScanResult) Flagged() []ScanItem {
    var out []ScanItem
    for _, item := range r.Items {
        if item.DepthFlag != nil && item.DepthFlag.Flagged {
            out = append(out, item)
        }
    }
    return out
}

const scanSystemPrompt = `You are a market intelligence agent. Analyze the problem and identify relevant tools, frameworks, and competitors.
Respond ONLY with valid JSON — no markdown, no explanation.`

const scanUserPromptTpl = `Scan the market for tools relevant to this problem.

PROBLEM: %s
JOBS: %s

Find 5–8 relevant tools/frameworks/products. For each, assess what it covers and its key gap.

Return JSON:
{"items":[{"name":"string","disposition":"ADOPT|EXTRACT|INSPIRE|MONITOR|IGNORE","covers_phases":["frame|hypothesize|scan|validate|experiment"],"key_strength":"string","key_gap":"string","stars":0,"primary_source_read":false,"architecture_reviewed":false,"desc_sentences":3,"source_count":1,"multi_source":false,"disposition_confidence":0.5}],"whitespace":"string describing the gap nobody fills","recommended_stack":["string"]}`

func Scan(ctx context.Context, c *LLMClient, frame *FrameResult) (*ScanResult, error) {
    jobs := strings.Join(frame.Jobs, "; ")
    resp, err := c.Chat(ctx, ChatRequest{
        Model:       DefaultSynthModel,
        Messages: []Message{
            {Role: "system", Content: scanSystemPrompt},
            {Role: "user", Content: fmt.Sprintf(scanUserPromptTpl, frame.ProblemStatement, jobs)},
        },
        MaxTokens:   2000,
        Temperature: 0.1,
    })
    if err != nil {
        return nil, fmt.Errorf("scan llm: %w", err)
    }
    content := strings.TrimSpace(resp.Content)
    // strip markdown fences if model disobeyed
    if strings.HasPrefix(content, "```") {
        lines := strings.Split(content, "\n")
        if len(lines) > 2 {
            content = strings.Join(lines[1:len(lines)-1], "\n")
        }
    }
    var raw struct {
        Items            []ScanItem `json:"items"`
        Whitespace       string     `json:"whitespace"`
        RecommendedStack []string   `json:"recommended_stack"`
    }
    if err := json.Unmarshal([]byte(content), &raw); err != nil {
        return nil, fmt.Errorf("scan parse (finish=%s): %w\ncontent: %s",
            resp.FinishReason, err, content)
    }

    // Apply depth evaluation to each item
    for i := range raw.Items {
        score := CoverageScore(raw.Items[i])
        raw.Items[i].CoverageScore = score
        flag := EvalDepth(raw.Items[i])
        if flag.Flagged {
            raw.Items[i].DepthFlag = &flag
        }
    }

    return &ScanResult{
        Items:            raw.Items,
        Whitespace:       raw.Whitespace,
        RecommendedStack: raw.RecommendedStack,
        CostUSD:          resp.CostUSD,
    }, nil
}
```

**Step 4: Run test**

```bash
source .env && go test ./internal/discovery/... -run TestScan -v -timeout 60s
```
Expected: PASS — items returned, some flagged

**Step 5: Commit**

```bash
git add internal/discovery/scan.go internal/discovery/scan_test.go
git commit -m "feat(discovery): Phase 3 SCAN with depth signal evaluation"
```

---

## Task 5: Checkpoint renderer

**Files:**
- Create: `internal/discovery/checkpoint.go`
- Create: `internal/discovery/checkpoint_test.go`

**Step 1: Write the failing test**

```go
// internal/discovery/checkpoint_test.go
package discovery_test

import (
    "strings"
    "testing"
    "github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestCheckpointRender_TwoSections(t *testing.T) {
    items := []discovery.ScanItem{
        {Name: "SettledTool", Disposition: discovery.DispositionInspire,
         CoverageScore: 0.8, PrimarySourceRead: true, ArchitectureReviewed: true,
         DescSentences: 12, MultiSource: true},
        {Name: "FlaggedTool", Disposition: discovery.DispositionExtract,
         CoverageScore: 0.18, PrimarySourceRead: false, Stars: 50000, DescSentences: 3,
         DepthFlag: &discovery.DepthFlag{Flagged: true, Reason: "no_primary_source",
             RecommendedAction: "deep_dive", Blocking: true}},
    }
    result := &discovery.ScanResult{Items: items, Whitespace: "nobody covers full pipeline"}
    out := discovery.RenderCheckpoint(result)

    if !strings.Contains(out, "Section A") {
        t.Error("missing Section A")
    }
    if !strings.Contains(out, "Section B") {
        t.Error("missing Section B")
    }
    if !strings.Contains(out, "FlaggedTool") {
        t.Error("FlaggedTool not in Section B")
    }
    if !strings.Contains(out, "[D]") {
        t.Error("missing deep-dive option")
    }
}
```

**Step 2: Run to confirm fail**

```bash
go test ./internal/discovery/... -run TestCheckpoint -v
```

**Step 3: Implement**

```go
// internal/discovery/checkpoint.go
package discovery

import (
    "fmt"
    "strings"
)

func RenderCheckpoint(result *ScanResult) string {
    var b strings.Builder
    settled := result.Settled()
    flagged := result.Flagged()

    fmt.Fprintf(&b, "\n╔══════════════════════════════════════════════════════════╗\n")
    fmt.Fprintf(&b, "  SCAN CHECKPOINT\n")
    fmt.Fprintf(&b, "  %d settled · %d flagged · whitespace: %s\n",
        len(settled), len(flagged), result.Whitespace)
    fmt.Fprintf(&b, "╚══════════════════════════════════════════════════════════╝\n")

    if len(settled) > 0 {
        fmt.Fprintf(&b, "\n── Section A — Settled (coverage ≥ 0.4, no flags) ──\n\n")
        for _, item := range settled {
            icon := dispositionIcon(item.Disposition)
            fmt.Fprintf(&b, "  %s %-30s coverage=%.2f  [%s]\n",
                icon, item.Name, item.CoverageScore, item.Disposition)
            if item.KeyStrength != "" {
                fmt.Fprintf(&b, "     strength: %s\n", item.KeyStrength)
            }
            if item.KeyGap != "" {
                fmt.Fprintf(&b, "     gap:      %s\n", item.KeyGap)
            }
        }
    }

    if len(flagged) > 0 {
        fmt.Fprintf(&b, "\n── Section B — Flagged (require depth decision) ──\n")
        for i, item := range flagged {
            fmt.Fprintf(&b, "\n  %d. %s", i+1, item.Name)
            if item.Stars > 0 {
                fmt.Fprintf(&b, " (%d★)", item.Stars)
            }
            fmt.Fprintf(&b, "\n")
            fmt.Fprintf(&b, "     coverage: %.2f/1.0 · disposition: %s\n",
                item.CoverageScore, item.Disposition)
            fmt.Fprintf(&b, "     reason:   %s\n", item.DepthFlag.Reason)
            if item.DepthFlag.Blocking {
                fmt.Fprintf(&b, "     ⚠️  BLOCKING — pipeline paused until resolved\n")
            }
            fmt.Fprintf(&b, "     options:\n")
            fmt.Fprintf(&b, "       [D] Deep dive now\n")
            fmt.Fprintf(&b, "       [P] Proceed provisional (tagged sdp:scan:unverified)\n")
            fmt.Fprintf(&b, "       [I] Downgrade to MONITOR\n")
        }
    }

    if len(result.RecommendedStack) > 0 {
        fmt.Fprintf(&b, "\n── Recommended stack ──\n")
        for _, s := range result.RecommendedStack {
            fmt.Fprintf(&b, "  • %s\n", s)
        }
    }

    return b.String()
}

func dispositionIcon(d Disposition) string {
    switch d {
    case DispositionAdopt:   return "🟢"
    case DispositionExtract: return "🔵"
    case DispositionInspire: return "💡"
    case DispositionMonitor: return "👁️ "
    case DispositionIgnore:  return "⬛"
    default:                 return "  "
    }
}
```

**Step 4: Run test**

```bash
go test ./internal/discovery/... -run TestCheckpoint -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/discovery/checkpoint.go internal/discovery/checkpoint_test.go
git commit -m "feat(discovery): checkpoint renderer — two-section output with depth flags"
```

---

## Task 6: Artifact writer

**Files:**
- Create: `internal/discovery/artifacts.go`
- Create: `internal/discovery/artifacts_test.go`

**Step 1: Write the failing test**

```go
// internal/discovery/artifacts_test.go
package discovery_test

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
    "github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestArtifacts_WritesFiles(t *testing.T) {
    dir := t.TempDir()
    session := &discovery.Session{
        Slug:  "test-idea",
        Date:  "2026-04-08",
        Frame: &discovery.FrameResult{
            RawIdea:          "test idea",
            ProblemStatement: "test problem",
            Jobs:             []string{"job 1"},
            Appetite:         "small",
        },
        Scan: &discovery.ScanResult{
            Items:      []discovery.ScanItem{{Name: "ToolA", Disposition: discovery.DispositionInspire}},
            Whitespace: "gap description",
        },
    }
    if err := discovery.WriteArtifacts(dir, session); err != nil {
        t.Fatalf("write: %v", err)
    }
    frameFile := filepath.Join(dir, "2026-04-08-test-idea-frame.md")
    if _, err := os.Stat(frameFile); err != nil {
        t.Errorf("frame file not created: %v", err)
    }
    content, _ := os.ReadFile(frameFile)
    if !strings.Contains(string(content), "test problem") {
        t.Error("frame file missing problem statement")
    }
}
```

**Step 2: Run to confirm fail**

```bash
go test ./internal/discovery/... -run TestArtifacts -v
```

**Step 3: Implement**

```go
// internal/discovery/artifacts.go
package discovery

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// Session holds all pipeline state for one discovery run.
type Session struct {
    Slug  string
    Date  string
    Frame *FrameResult
    Scan  *ScanResult
}

func NewSession(idea string) *Session {
    slug := slugify(idea)
    return &Session{
        Slug: slug,
        Date: time.Now().Format("2006-01-02"),
    }
}

func slugify(s string) string {
    s = strings.ToLower(s)
    var b strings.Builder
    for _, r := range s {
        if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
            b.WriteRune(r)
        } else if r == ' ' || r == '-' {
            b.WriteRune('-')
        }
    }
    slug := b.String()
    if len(slug) > 40 {
        slug = slug[:40]
    }
    return strings.Trim(slug, "-")
}

func WriteArtifacts(dir string, s *Session) error {
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return fmt.Errorf("mkdir: %w", err)
    }
    prefix := filepath.Join(dir, s.Date+"-"+s.Slug)

    if s.Frame != nil {
        if err := writeFrame(prefix+"-frame.md", s.Frame); err != nil {
            return err
        }
    }
    if s.Scan != nil {
        if err := writeScan(prefix+"-scan.md", s.Scan); err != nil {
            return err
        }
    }
    return nil
}

func writeFrame(path string, f *FrameResult) error {
    var b strings.Builder
    fmt.Fprintf(&b, "# Discovery Frame\n\n")
    fmt.Fprintf(&b, "**Raw idea:** %s\n\n", f.RawIdea)
    fmt.Fprintf(&b, "## Problem Statement\n\n%s\n\n", f.ProblemStatement)
    fmt.Fprintf(&b, "## Jobs to Be Done\n\n")
    for _, j := range f.Jobs {
        fmt.Fprintf(&b, "- %s\n", j)
    }
    fmt.Fprintf(&b, "\n**Appetite:** %s\n\n**Scope:** %s\n", f.Appetite, f.Scope)
    return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeScan(path string, r *ScanResult) error {
    var b strings.Builder
    fmt.Fprintf(&b, "# Discovery Scan\n\n")
    fmt.Fprintf(&b, "**Whitespace:** %s\n\n", r.Whitespace)
    fmt.Fprintf(&b, "## Landscape\n\n")
    fmt.Fprintf(&b, "| Tool | Disposition | Coverage | Flagged |\n")
    fmt.Fprintf(&b, "|---|---|---|---|\n")
    for _, item := range r.Items {
        flagged := "—"
        if item.DepthFlag != nil && item.DepthFlag.Flagged {
            flagged = "⚠️ " + item.DepthFlag.Reason
        }
        fmt.Fprintf(&b, "| %s | %s | %.2f | %s |\n",
            item.Name, item.Disposition, item.CoverageScore, flagged)
    }
    // Append raw JSON for downstream use
    raw, _ := json.MarshalIndent(r, "", "  ")
    fmt.Fprintf(&b, "\n```json\n%s\n```\n", raw)
    return os.WriteFile(path, []byte(b.String()), 0o644)
}
```

**Step 4: Run test**

```bash
go test ./internal/discovery/... -run TestArtifacts -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/discovery/artifacts.go internal/discovery/artifacts_test.go
git commit -m "feat(discovery): artifact writer — frame + scan markdown files"
```

---

## Task 7: CLI command `sdp discover`

**Files:**
- Create: `cmd/sdp/cmd_discover.go`
- Modify: `cmd/sdp/main.go` — add `case "discover":`

**Step 1: No test (CLI integration — covered by manual smoke test in step 4)**

**Step 2: Implement**

```go
// cmd/sdp/cmd_discover.go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "path/filepath"

    "github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func runDiscover(args []string) {
    fs := flag.NewFlagSet("discover", flag.ExitOnError)
    outDir  := fs.String("out", "docs/discovery", "output directory for artifacts")
    model   := fs.String("model", "", "override default LLM model")
    _ = fs.Parse(args)

    if fs.NArg() < 1 {
        fmt.Fprintln(os.Stderr, "usage: sdp discover [--out DIR] \"raw idea\"")
        os.Exit(2)
    }
    idea := fs.Arg(0)

    apiKey := os.Getenv("OPENROUTER_API_KEY")
    if apiKey == "" {
        fmt.Fprintln(os.Stderr, "error: OPENROUTER_API_KEY not set")
        os.Exit(1)
    }

    client := discovery.NewLLMClient(apiKey, discovery.DefaultOpenRouterBase)
    if *model != "" {
        discovery.DefaultPlannerModel = *model
        discovery.DefaultSynthModel   = *model
    }

    ctx := context.Background()
    session := discovery.NewSession(idea)

    // ── Phase 1: FRAME ─────────────────────────────────────────────
    fmt.Printf("🔍 Phase 1: Framing idea...\n")
    frame, err := discovery.Frame(ctx, client, idea)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: frame: %v\n", err)
        os.Exit(1)
    }
    session.Frame = frame
    fmt.Printf("   problem: %s\n", frame.ProblemStatement)
    fmt.Printf("   appetite: %s\n\n", frame.Appetite)

    // ── Phase 3: SCAN ──────────────────────────────────────────────
    fmt.Printf("🔍 Phase 3: Scanning market...\n")
    scanResult, err := discovery.Scan(ctx, client, frame)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: scan: %v\n", err)
        os.Exit(1)
    }
    session.Scan = scanResult
    fmt.Printf("   found %d items (settled=%d flagged=%d)\n",
        len(scanResult.Items),
        len(scanResult.Settled()),
        len(scanResult.Flagged()))
    fmt.Printf("   cost: $%.5f\n\n", scanResult.CostUSD)

    // ── Checkpoint ─────────────────────────────────────────────────
    fmt.Println(discovery.RenderCheckpoint(scanResult))

    // ── Write artifacts ────────────────────────────────────────────
    absOut, _ := filepath.Abs(*outDir)
    if err := discovery.WriteArtifacts(absOut, session); err != nil {
        fmt.Fprintf(os.Stderr, "error: write artifacts: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("📁 Artifacts written to %s/\n", absOut)

    // ── Create beads issue ─────────────────────────────────────────
    fmt.Printf("\n📌 Creating beads issue...\n")
    issueID, err := createDiscoveryIssue(idea, frame, absOut, session)
    if err != nil {
        fmt.Fprintf(os.Stderr, "warning: beads issue: %v\n", err)
    } else {
        fmt.Printf("   created: %s\n", issueID)
    }
}
```

**Step 3: Add beads helper and wire into main**

```go
// In cmd/sdp/cmd_discover.go (append)

import "os/exec"

func createDiscoveryIssue(idea string, frame *discovery.FrameResult,
    artifactDir string, session *discovery.Session) (string, error) {

    desc := fmt.Sprintf("## Discovery: %s\n\n**Problem:** %s\n\n**Appetite:** %s\n\n**Artifacts:** %s/",
        idea, frame.ProblemStatement, frame.Appetite, artifactDir)

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
    // bd outputs the issue ID on stdout
    return string(out), nil
}
```

Add to `cmd/sdp/main.go` switch (after `case "plan":`):
```go
case "discover":
    runDiscover(os.Args[2:])
```

**Step 4: Build and smoke test**

```bash
go build ./cmd/sdp/...
source .env && ./sdp discover "automate product discovery using AI agents"
```
Expected:
```
🔍 Phase 1: Framing idea...
   problem: ...
🔍 Phase 3: Scanning market...
   found N items (settled=X flagged=Y)
╔══...SCAN CHECKPOINT...
📁 Artifacts written to docs/discovery/
📌 Creating beads issue...
```

**Step 5: Commit**

```bash
git add cmd/sdp/cmd_discover.go cmd/sdp/main.go
git commit -m "feat(discovery): sdp discover command — Phase 1+3 with checkpoint"
```

---

## Task 8: Run full pipeline on 3 real ideas (dogfood)

**Goal:** Validate depth signal heuristics against real inputs before building Phase 2 and 4.

**Step 1: Run on three ideas**

```bash
source .env

./sdp discover "automate product discovery using AI agents"
./sdp discover "build a CLI tool that monitors competitor pricing in real time"
./sdp discover "replace standups with async AI-generated team summaries"
```

**Step 2: Check depth signal quality**

For each run, verify:
- Section B items are genuinely under-researched (not false positives)
- Section A items are genuinely settled (not false negatives)
- Coverage scores feel calibrated

**Step 3: Record findings in beads**

```bash
bd update <issue-id> --notes "dogfood run 1: N flagged, M settled. Heuristic H1 fired on X — correct/incorrect. ..."
```

**Step 4: Adjust heuristics if needed**

Edit `internal/discovery/depth.go` thresholds based on findings. Re-run tests.

**Step 5: Commit findings**

```bash
git add internal/discovery/depth.go
git commit -m "fix(discovery): calibrate depth heuristics from dogfood run"
```

---

## What's NOT in this plan (next iterations)

- Phase 2 HYPOTHESIZE (Strategyzer Test Card, RAT ranking)
- Phase 4a desk research validation (claims → evidence for/against)
- Phase 4b experiment design
- GPT Researcher integration (get_research_context)
- MCP fan-out (Exa, Brave, HN, Reddit, GitHub)
- Async checkpoint (beads + notify when offline)
- Go loop detection middleware (from DeerFlow INSPIRE)

Each of these is a separate plan once Phase 1+3 prototype is validated.

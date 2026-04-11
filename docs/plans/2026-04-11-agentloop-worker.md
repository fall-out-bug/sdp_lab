# SDP Mini-Harness: Worker Layer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the worker layer of the SDP Mini-Harness (`internal/agentloop`) using strict TDD. Picks up after the foundation layer (Tasks 1–5). After all five tasks here pass, the package has: a ModelGateway interface with StubGateway test double, `executeCalls` + `makeCompletionSignalTool`, a complete `Run()` loop, a GateEngine wrapping `harness.EvaluateCompliance`, and a PhaseRouter + ToolRegistry.

**Architecture:** The worker layer sits between the foundation types/store and the Harness orchestrator. `ModelGateway` abstracts LLM calls. `executeCalls` runs tools in parallel goroutines. `Run()` drives one full phase-turn: trim → LLM call → drain events → execute tools → loop. `GateEngine` wraps `harness.EvaluateCompliance` (which takes NO context) with a 5 s circuit breaker. `PhaseRouter` resolves models, builds `LoopConfig` (injecting `completion_signal`), and navigates phase transitions. `ToolRegistry` maintains the per-phase allowlist.

**Tech Stack:**
- Go module: `sdp_dev`, go 1.26
- Target package: `sdp_dev/internal/agentloop`
- Test assertions: `github.com/stretchr/testify v1.11.1`
- Existing: `sdp_dev/internal/harness` — `EvaluateCompliance(contract *TaskContract, snapshot *TaskSnapshot) ComplianceReport` (no context parameter)
- Test runner: `go test ./internal/agentloop/... -race`

**Key correctness rules (from v14 design):**
- **Y1**: `Loop` emits `Event{Type:"tool_end", ToolID: result.ID, ...}` — `ToolID` must match the original `ToolCall.ID`
- **N6**: `completion_signal` is added by `BuildLoopConfig`, NOT by `ToolRegistry` — never appears in `PhaseConfig.Tools`
- **Fix W2/X1**: `NewPhaseRouter` takes `cm ContextManager` (can be nil); `BuildLoopConfig` wires it into `LoopConfig.ContextManager`
- **BeforeToolCall** is called before `Execute`; **AfterToolCall** is called even for rejected calls and tool-not-found cases
- **EvaluateCompliance** has signature `(contract, snapshot)` — no context. GateEngine wraps it in a goroutine, uses `select` on result channel vs `evalCtx.Done()`
- **Gateway in LoopConfig**: `Run()` receives `cfg LoopConfig` which includes `Gateway ModelGateway` field (added here)

---

### Task 6: ModelGateway interface + StubGateway

**Files:**
- Create: `internal/agentloop/gateway.go`
- Create: `internal/agentloop/gateway_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/gateway_test.go`:

```go
package agentloop

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelGateway_interfaceSatisfaction verifies StubGateway satisfies ModelGateway at compile time.
func TestModelGateway_interfaceSatisfaction(t *testing.T) {
	var _ ModelGateway = (*StubGateway)(nil)
}

// TestStubGateway_isAvailable returns true for registered models, false otherwise.
func TestStubGateway_isAvailable(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{{Type: "done"}})

	assert.True(t, sg.IsAvailable("gpt-4.1"), "registered model must be available")
	assert.False(t, sg.IsAvailable("nonexistent-model"), "unknown model must not be available")
}

// TestStubGateway_call_returnsScriptedEvents verifies the event channel delivers scripted events.
func TestStubGateway_call_returnsScriptedEvents(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{
		{Type: "text_delta", Delta: "hello"},
		{Type: "done"},
	})

	cfg := LoopConfig{Model: "gpt-4.1"}
	ch, err := sg.Call(context.Background(), []Message{{Role: "user", Content: "hi"}}, cfg)
	require.NoError(t, err)

	var events []Event
	for ev := range ch {
		events = append(events, ev)
	}
	require.Len(t, events, 2)
	assert.Equal(t, "text_delta", events[0].Type)
	assert.Equal(t, "hello", events[0].Delta)
	assert.Equal(t, "done", events[1].Type)
}

// TestStubGateway_recordsCalls verifies ModelCall recording for assertion in tests.
func TestStubGateway_recordsCalls(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{{Type: "done"}})

	cfg := LoopConfig{Model: "gpt-4.1", SystemPrompt: "be helpful"}
	msgs := []Message{{Role: "user", Content: "build it"}}
	ch, err := sg.Call(context.Background(), msgs, cfg)
	require.NoError(t, err)
	// drain
	for range ch {
	}

	require.Len(t, sg.Calls, 1)
	assert.Equal(t, "gpt-4.1", sg.Calls[0].Model)
	assert.Equal(t, msgs, sg.Calls[0].Messages)
}

// TestStubGateway_call_unknownModel returns error for unregistered model.
func TestStubGateway_call_unknownModel(t *testing.T) {
	sg := NewStubGateway()
	_, err := sg.Call(context.Background(), nil, LoopConfig{Model: "unknown"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

// TestStubGateway_addResponse_multipleModels supports independent scripted responses per model.
func TestStubGateway_addResponse_multipleModels(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("model-a", []Event{{Type: "text_delta", Delta: "from A"}, {Type: "done"}})
	sg.AddResponse("model-b", []Event{{Type: "text_delta", Delta: "from B"}, {Type: "done"}})

	assert.True(t, sg.IsAvailable("model-a"))
	assert.True(t, sg.IsAvailable("model-b"))

	chA, err := sg.Call(context.Background(), nil, LoopConfig{Model: "model-a"})
	require.NoError(t, err)
	var evA []Event
	for ev := range chA {
		evA = append(evA, ev)
	}
	assert.Equal(t, "from A", evA[0].Delta)
}
```

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestModelGateway|TestStubGateway" -v
```
Expected: FAIL — `ModelGateway`, `StubGateway`, `ModelCall`, `NewStubGateway` undefined.

**Step 3: Write minimal implementation**

Create `internal/agentloop/gateway.go`:

```go
package agentloop

import (
	"context"
	"fmt"
)

// ModelGateway abstracts LLM provider calls. Run() uses this to stream events.
// Production implementations wrap OpenRouter / Anthropic / OpenAI SDKs.
// Tests use StubGateway.
type ModelGateway interface {
	// Call initiates a streaming LLM request and returns a channel of Events.
	// The channel is closed when the response is complete (after "done" event) or on error.
	// cfg.Model determines which model to call.
	Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error)

	// IsAvailable returns true if the given model name can be routed.
	// PhaseRouter uses this to pick the first available model from PhaseConfig.Models.
	IsAvailable(model string) bool
}

// ModelCall records a single Call() invocation for test assertions.
type ModelCall struct {
	Model    string
	Messages []Message
	Config   LoopConfig
}

// StubGateway is an in-memory test double for ModelGateway.
// Register scripted Event sequences with AddResponse; recorded calls are in Calls.
type StubGateway struct {
	responses map[string][]Event
	Calls     []ModelCall
}

// NewStubGateway creates an initialized StubGateway.
func NewStubGateway() *StubGateway {
	return &StubGateway{
		responses: make(map[string][]Event),
	}
}

// AddResponse registers a scripted Event sequence for model.
// Call() will emit these events in order, then close the channel.
// Calling AddResponse multiple times for the same model overwrites the previous sequence.
func (sg *StubGateway) AddResponse(model string, events []Event) {
	sg.responses[model] = events
}

// IsAvailable returns true if model has a registered response sequence.
func (sg *StubGateway) IsAvailable(model string) bool {
	_, ok := sg.responses[model]
	return ok
}

// Call records the invocation, then returns a channel that emits the scripted events.
// Returns an error immediately if model has no registered response.
func (sg *StubGateway) Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
	events, ok := sg.responses[cfg.Model]
	if !ok {
		return nil, fmt.Errorf("StubGateway: unknown model %q — use AddResponse to register it", cfg.Model)
	}

	sg.Calls = append(sg.Calls, ModelCall{
		Model:    cfg.Model,
		Messages: msgs,
		Config:   cfg,
	})

	ch := make(chan Event, len(events))
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestModelGateway|TestStubGateway" -race -v
```
Expected: PASS — all 6 gateway tests green.

**Step 5: Commit**

```
git add internal/agentloop/gateway.go internal/agentloop/gateway_test.go
git commit -m "feat(agentloop): Task 6 — ModelGateway interface + StubGateway test double (records calls, scripted events, IsAvailable)"
```

---

### Task 7: executeCalls + findTool + makeCompletionSignalTool

**Files:**
- Create: `internal/agentloop/loop.go` (partial — executeCalls, findTool, makeCompletionSignalTool only; Run() added in Task 8)
- Create: `internal/agentloop/loop_execute_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/loop_execute_test.go`:

```go
package agentloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- helpers ----

func makeTool(name string, out string, err error) Tool {
	return Tool{
		Name: name,
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			return out, err
		},
	}
}

// TestExecuteCalls_success: two parallel tools, both succeed; results match.
func TestExecuteCalls_success(t *testing.T) {
	tools := []Tool{
		makeTool("search", "result1", nil),
		makeTool("read_file", "content", nil),
	}

	calls := []ToolCall{
		{ID: "tc1", Name: "search", Arguments: json.RawMessage(`{}`)},
		{ID: "tc2", Name: "read_file", Arguments: json.RawMessage(`{}`)},
	}

	cfg := LoopConfig{} // no hooks
	results := executeCalls(context.Background(), calls, tools, cfg)

	require.Len(t, results, 2)
	// Results are in the same order as calls (indexed assignment in goroutines).
	assert.Equal(t, "tc1", results[0].ID)
	assert.Equal(t, "search", results[0].Name)
	assert.Equal(t, "result1", results[0].Output)
	assert.Nil(t, results[0].Err)

	assert.Equal(t, "tc2", results[1].ID)
	assert.Equal(t, "read_file", results[1].Name)
	assert.Equal(t, "content", results[1].Output)
	assert.Nil(t, results[1].Err)
}

// TestExecuteCalls_preservesArguments: ToolResult.Arguments matches original ToolCall.Arguments (Fix T1).
func TestExecuteCalls_preservesArguments(t *testing.T) {
	args := json.RawMessage(`{"cmd":"go test"}`)
	tools := []Tool{makeTool("bash", "PASS", nil)}
	calls := []ToolCall{{ID: "tc1", Name: "bash", Arguments: args}}

	results := executeCalls(context.Background(), calls, tools, LoopConfig{})
	require.Len(t, results, 1)
	assert.Equal(t, string(args), string(results[0].Arguments), "Fix T1: Arguments must be preserved")
}

// TestExecuteCalls_beforeHookReject: BeforeToolCall returns error → ToolResult.Err set, AfterToolCall still called.
func TestExecuteCalls_beforeHookReject(t *testing.T) {
	tools := []Tool{makeTool("bash", "should not run", nil)}
	calls := []ToolCall{{ID: "tc1", Name: "bash", Arguments: json.RawMessage(`{}`)}}

	var afterCalled bool
	cfg := LoopConfig{
		BeforeToolCall: func(name string, args json.RawMessage) error {
			return errors.New("hook rejected: tool not allowed in this phase")
		},
		AfterToolCall: func(r ToolResult) error {
			afterCalled = true
			return nil
		},
	}

	results := executeCalls(context.Background(), calls, tools, cfg)

	require.Len(t, results, 1)
	require.Error(t, results[0].Err, "BeforeToolCall rejection must set ToolResult.Err")
	assert.Contains(t, results[0].Err.Error(), "before hook rejected")
	assert.True(t, afterCalled, "AfterToolCall must be called even for rejected tool calls (Fix A5)")
}

// TestExecuteCalls_toolNotFound: tool name not in slice → Err="tool not in phase allowlist", AfterToolCall still called.
func TestExecuteCalls_toolNotFound(t *testing.T) {
	tools := []Tool{makeTool("allowed_tool", "ok", nil)}
	calls := []ToolCall{{ID: "tc1", Name: "forbidden_tool", Arguments: json.RawMessage(`{}`)}}

	var afterCalled bool
	var afterResult ToolResult
	cfg := LoopConfig{
		AfterToolCall: func(r ToolResult) error {
			afterCalled = true
			afterResult = r
			return nil
		},
	}

	results := executeCalls(context.Background(), calls, tools, cfg)

	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	assert.Contains(t, results[0].Err.Error(), "tool not in phase allowlist")
	assert.True(t, afterCalled, "AfterToolCall must be called even when tool is not found")
	assert.Equal(t, "forbidden_tool", afterResult.Name)
}

// TestExecuteCalls_afterCallbackError: AfterToolCall error gets wrapped into ToolResult.Err (Fix A4).
func TestExecuteCalls_afterCallbackError(t *testing.T) {
	tools := []Tool{makeTool("bash", "ok", nil)}
	calls := []ToolCall{{ID: "tc1", Name: "bash", Arguments: json.RawMessage(`{}`)}}

	cfg := LoopConfig{
		AfterToolCall: func(r ToolResult) error {
			return errors.New("callback failed")
		},
	}

	results := executeCalls(context.Background(), calls, tools, cfg)
	require.Len(t, results, 1)
	require.Error(t, results[0].Err, "AfterToolCall error must be wrapped into ToolResult.Err (Fix A4)")
	assert.Contains(t, results[0].Err.Error(), "callback")
}

// TestExecuteCalls_parallelExecution: goroutines truly run in parallel (no serial dependency).
func TestExecuteCalls_parallelExecution(t *testing.T) {
	var mu sync.Mutex
	started := make([]string, 0, 2)

	// Both tools block on a channel until both have started — proves parallel execution.
	gate := make(chan struct{})

	makeBlockingTool := func(name string) Tool {
		return Tool{
			Name: name,
			Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
				mu.Lock()
				started = append(started, name)
				allStarted := len(started) == 2
				mu.Unlock()
				if allStarted {
					close(gate)
				}
				<-gate // wait until both are started
				return name + "_done", nil
			},
		}
	}

	tools := []Tool{makeBlockingTool("tool1"), makeBlockingTool("tool2")}
	calls := []ToolCall{
		{ID: "tc1", Name: "tool1", Arguments: json.RawMessage(`{}`)},
		{ID: "tc2", Name: "tool2", Arguments: json.RawMessage(`{}`)},
	}

	results := executeCalls(context.Background(), calls, tools, LoopConfig{})
	require.Len(t, results, 2)
	assert.Nil(t, results[0].Err)
	assert.Nil(t, results[1].Err)
}

// TestMakeCompletionSignalTool: Execute sets flag.signaled=true and flag.summary from args.
func TestMakeCompletionSignalTool(t *testing.T) {
	flag := &completionFlag{}
	tool := makeCompletionSignalTool(flag)

	assert.Equal(t, "completion_signal", tool.Name)

	args := json.RawMessage(`{"summary":"discovered 3 competitors"}`)
	out, err := tool.Execute(context.Background(), "tc-signal", args)
	require.NoError(t, err)
	assert.NotEmpty(t, out, "completion_signal Execute must return a non-empty acknowledgement string")

	flag.mu.Lock()
	defer flag.mu.Unlock()
	assert.True(t, flag.signaled, "Execute must set flag.signaled = true")
	assert.Equal(t, "discovered 3 competitors", flag.summary, "Execute must capture summary from args")
}

// TestMakeCompletionSignalTool_emptySummary: handles missing summary field gracefully.
func TestMakeCompletionSignalTool_emptySummary(t *testing.T) {
	flag := &completionFlag{}
	tool := makeCompletionSignalTool(flag)

	_, err := tool.Execute(context.Background(), "tc-signal", json.RawMessage(`{}`))
	require.NoError(t, err)

	flag.mu.Lock()
	defer flag.mu.Unlock()
	assert.True(t, flag.signaled)
	assert.Equal(t, "", flag.summary)
}

// TestFindTool: returns correct tool by name and false when not present.
func TestFindTool(t *testing.T) {
	tools := []Tool{
		{Name: "bash"},
		{Name: "read_file"},
	}

	got, ok := findTool(tools, "bash")
	assert.True(t, ok)
	assert.Equal(t, "bash", got.Name)

	_, ok2 := findTool(tools, "nonexistent")
	assert.False(t, ok2)
}

// TestExecuteCalls_afterCallbackError_onRejected: AfterToolCall error on rejected call wraps both errors.
func TestExecuteCalls_afterCallbackError_onRejected(t *testing.T) {
	tools := []Tool{makeTool("bash", "ok", nil)}
	calls := []ToolCall{{ID: "tc1", Name: "bash", Arguments: json.RawMessage(`{}`)}}

	cfg := LoopConfig{
		BeforeToolCall: func(name string, args json.RawMessage) error {
			return fmt.Errorf("hook rejected")
		},
		AfterToolCall: func(r ToolResult) error {
			return fmt.Errorf("callback also failed")
		},
	}

	results := executeCalls(context.Background(), calls, tools, cfg)
	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	// Both errors must be present in the message.
	errMsg := results[0].Err.Error()
	assert.Contains(t, errMsg, "before hook rejected")
	assert.Contains(t, errMsg, "callback")
}
```

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestExecuteCalls|TestMakeCompletionSignalTool|TestFindTool" -v
```
Expected: FAIL — `executeCalls`, `findTool`, `makeCompletionSignalTool` undefined.

**Step 3: Write minimal implementation**

Create `internal/agentloop/loop.go` (partial — no `Run()` yet):

```go
package agentloop

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// findTool returns the Tool with the given name from the slice, or (zero, false).
func findTool(tools []Tool, name string) (Tool, bool) {
	for _, t := range tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

// executeCalls executes tool calls in parallel goroutines.
//
// Key contracts from v14 design:
//   - Fix A5: BeforeToolCall is called BEFORE Execute; error = rejection (Execute is skipped).
//   - Fix A4: AfterToolCall error is NOT ignored — it is wrapped into ToolResult.Err.
//   - AfterToolCall is called even for rejected and tool-not-found cases (evidence must know about failures).
//   - AfterToolCall is called SYNCHRONOUSLY before wg.Done() — all callbacks complete before wg.Wait() returns.
//   - Results slice is indexed by call position — order matches input calls slice.
//   - Fix T1: ToolResult.Arguments is set from ToolCall.Arguments.
func executeCalls(ctx context.Context, calls []ToolCall, tools []Tool, cfg LoopConfig) []ToolResult {
	results := make([]ToolResult, len(calls))
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(i int, call ToolCall) {
			defer wg.Done()

			// Fix A5: BeforeToolCall — pre-hook; error = call rejected, Execute is skipped.
			if cfg.BeforeToolCall != nil {
				if err := cfg.BeforeToolCall(call.Name, call.Arguments); err != nil {
					results[i] = ToolResult{
						ID:        call.ID,
						Name:      call.Name,
						Arguments: call.Arguments,
						Err:       fmt.Errorf("before hook rejected: %w", err),
					}
					// AfterToolCall still called for rejected calls (evidence must see failures).
					if cfg.AfterToolCall != nil {
						if cbErr := cfg.AfterToolCall(results[i]); cbErr != nil {
							// Fix A4: wrap both errors — neither is silently dropped.
							results[i].Err = fmt.Errorf("%w; callback: %v", results[i].Err, cbErr)
						}
					}
					return
				}
			}

			// Find tool in allowlist.
			tool, ok := findTool(tools, call.Name)
			if !ok {
				results[i] = ToolResult{
					ID:        call.ID,
					Name:      call.Name,
					Arguments: call.Arguments,
					Err:       fmt.Errorf("tool not in phase allowlist: %q", call.Name),
				}
			} else {
				// Execute with per-tool timeout (30 s default).
				tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				out, err := tool.Execute(tctx, call.ID, call.Arguments)
				results[i] = ToolResult{
					ID:        call.ID,
					Name:      call.Name,
					Arguments: call.Arguments,
					Output:    out,
					Err:       err,
				}
			}

			// Fix A4: AfterToolCall error is NOT ignored.
			if cfg.AfterToolCall != nil {
				if cbErr := cfg.AfterToolCall(results[i]); cbErr != nil {
					results[i].Err = fmt.Errorf("callback: %w", cbErr)
				}
			}
		}(i, call)
	}

	wg.Wait() // all executions AND callbacks complete before returning
	return results
}

// makeCompletionSignalTool returns a Tool that sets flag.signaled=true and captures flag.summary.
// Fix R2-2: flag is an explicit pointer — no shared state via local vars, no race condition.
// BuildLoopConfig passes the flag; RunPhase reads it after draining the event channel.
// The Loop sees completion_signal as a normal tool call — it does NOT know about phases.
func makeCompletionSignalTool(flag *completionFlag) Tool {
	return Tool{
		Name:        "completion_signal",
		Description: "Signal that the current phase work is complete. Harness will run gate check and decide on transition.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"summary": {"type": "string", "description": "Brief summary of work completed in this phase"}
			},
			"required": ["summary"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			var a struct {
				Summary string `json:"summary"`
			}
			// Unmarshal is best-effort; missing summary is allowed (warned by RunPhase, not here).
			_ = json.Unmarshal(args, &a)

			flag.mu.Lock()
			flag.signaled = true
			flag.summary = a.Summary
			flag.mu.Unlock()

			return "completion noted — harness will evaluate gate", nil
		},
	}
}

// Run is defined in Task 8. Placeholder prevents "declared but not used" issues for
// the Gateway field added to LoopConfig — see LoopConfig.Gateway below.
// The Gateway field is added here so StubGateway tests in Task 6 continue to compile
// (LoopConfig is passed to StubGateway.Call; the field is ignored by StubGateway).
```

> **Implementation note:** `LoopConfig` currently lives in `types.go` (Task 1). The `Gateway ModelGateway` field must be added to it now. Update `types.go` before writing `loop.go`:

In `internal/agentloop/types.go`, add `Gateway ModelGateway` to `LoopConfig`:

```go
type LoopConfig struct {
    Model          string
    SystemPrompt   string
    Tools          []Tool
    MaxTokens      int
    TurnTimeout    time.Duration
    BeforeToolCall func(name string, args json.RawMessage) error
    AfterToolCall  func(result ToolResult) error
    ContextManager ContextManager
    Gateway        ModelGateway // used by Run() to make LLM calls
}
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestExecuteCalls|TestMakeCompletionSignalTool|TestFindTool" -race -v
```
Expected: PASS — all 11 tests green with race detector.

**Step 5: Commit**

```
git add internal/agentloop/loop.go internal/agentloop/loop_execute_test.go internal/agentloop/types.go
git commit -m "feat(agentloop): Task 7 — executeCalls (parallel, hooks, allowlist), findTool, makeCompletionSignalTool; Gateway field in LoopConfig"
```

---

### Task 8: Loop.Run()

**Files:**
- Edit: `internal/agentloop/loop.go` (add `Run()` implementation)
- Create: `internal/agentloop/loop_run_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/loop_run_test.go`:

```go
package agentloop

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- helpers ----

// collectEvents drains ch into a slice. Blocks until ch is closed.
func collectEvents(ch <-chan Event) []Event {
	var out []Event
	for ev := range ch {
		out = append(out, ev)
	}
	return out
}

// noToolCfg builds a LoopConfig with a StubGateway and no tools.
func noToolCfg(model string, sg *StubGateway) LoopConfig {
	return LoopConfig{
		Model:   model,
		Gateway: sg,
	}
}

// TestRun_noTools_emitsDone: single LLM response with no tool calls → "done" event emitted to caller.
func TestRun_noTools_emitsDone(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{
		{Type: "text_delta", Delta: "I found 3 competitors."},
		{Type: "done"},
	})

	cfg := noToolCfg("gpt-4.1", sg)
	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "discover"}}, cfg)
	require.NoError(t, err)

	events := collectEvents(ch)
	types := make([]string, len(events))
	for i, ev := range events {
		types[i] = ev.Type
	}

	assert.Contains(t, types, "text_delta")
	assert.Equal(t, "done", types[len(types)-1], "last event must be 'done'")
}

// TestRun_withTools_executesAndContinues: LLM calls 1 tool → tool executes → second LLM response → done.
func TestRun_withTools_executesAndContinues(t *testing.T) {
	sg := NewStubGateway()

	// First LLM call: requests a tool, then "done" (meaning this assistant turn ended)
	sg.AddResponse("gpt-4.1", []Event{
		{Type: "tool_call", ToolCalls: []ToolCall{
			{ID: "tc1", Name: "web_search", Arguments: json.RawMessage(`{"query":"competitors"}`)},
		}},
		{Type: "done"},
	})

	// Second LLM call (after tool result appended): final text response
	// StubGateway returns its sequence every time Call is invoked for the same model,
	// but here we need two different responses. Use a counter-based approach.
	callCount := 0
	sg2 := &countingGateway{
		responses: [][]Event{
			{
				{Type: "tool_call", ToolCalls: []ToolCall{
					{ID: "tc1", Name: "web_search", Arguments: json.RawMessage(`{"query":"competitors"}`)},
				}},
				{Type: "done"},
			},
			{
				{Type: "text_delta", Delta: "Found 3 competitors."},
				{Type: "done"},
			},
		},
		callCount: &callCount,
	}

	searchTool := Tool{
		Name: "web_search",
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			return "competitor1, competitor2, competitor3", nil
		},
	}

	cfg := LoopConfig{
		Model:   "gpt-4.1",
		Gateway: sg2,
		Tools:   []Tool{searchTool},
	}

	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "discover"}}, cfg)
	require.NoError(t, err)

	events := collectEvents(ch)
	require.True(t, len(events) > 0, "must emit at least one event")
	assert.Equal(t, "done", events[len(events)-1].Type, "last event must be 'done'")

	// Should have called gateway twice (initial + after tool result).
	assert.Equal(t, 2, callCount)

	// Must have emitted tool_end event.
	toolEndEvents := filterByType(events, "tool_end")
	require.Len(t, toolEndEvents, 1)
	assert.Equal(t, "tc1", toolEndEvents[0].ToolID, "Fix Y1: tool_end.ToolID must match ToolCall.ID")
	assert.Equal(t, "web_search", toolEndEvents[0].ToolName)
}

// TestRun_toolEndCarriesToolID: Fix Y1 — event.ToolID matches the original ToolCall.ID.
func TestRun_toolEndCarriesToolID(t *testing.T) {
	callCount := 0
	gw := &countingGateway{
		responses: [][]Event{
			{
				{Type: "tool_call", ToolCalls: []ToolCall{
					{ID: "specific-tool-id-42", Name: "read_file", Arguments: json.RawMessage(`{}`)},
				}},
				{Type: "done"},
			},
			{
				{Type: "text_delta", Delta: "file content processed"},
				{Type: "done"},
			},
		},
		callCount: &callCount,
	}

	readFileTool := Tool{
		Name: "read_file",
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			return "file contents", nil
		},
	}

	cfg := LoopConfig{
		Model:   "gpt-4.1",
		Gateway: gw,
		Tools:   []Tool{readFileTool},
	}

	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "read it"}}, cfg)
	require.NoError(t, err)
	events := collectEvents(ch)

	toolEndEvents := filterByType(events, "tool_end")
	require.Len(t, toolEndEvents, 1)
	assert.Equal(t, "specific-tool-id-42", toolEndEvents[0].ToolID,
		"Fix Y1: ToolID in 'tool_end' event must equal the original ToolCall.ID")
}

// TestRun_contextCancellation: cancel ctx → channel closes with an error event.
func TestRun_contextCancellation(t *testing.T) {
	// Gateway that blocks until context is cancelled.
	blockingGW := &blockingGateway{}

	cfg := LoopConfig{
		Model:   "gpt-4.1",
		Gateway: blockingGW,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	ch, err := Run(ctx, []Message{{Role: "user", Content: "go"}}, cfg)
	require.NoError(t, err, "Run() itself must not error immediately")

	events := collectEvents(ch)
	// Must receive at least one event; last event should be "error" or channel just closes.
	if len(events) > 0 {
		lastType := events[len(events)-1].Type
		// Accept either an "error" event or empty (channel closed without events on fast cancel).
		assert.True(t, lastType == "error" || lastType == "done",
			"on cancellation, last event must be 'error' or channel closes cleanly; got %q", lastType)
	}
	// Primary assertion: channel must be closed (collectEvents returned).
}

// TestRun_contextManagerTrimCalled: if ContextManager != nil → Trim is called before each LLM call.
func TestRun_contextManagerTrimCalled(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{
		{Type: "text_delta", Delta: "ok"},
		{Type: "done"},
	})

	trimCallCount := 0
	cm := &countingContextManager{count: &trimCallCount}

	cfg := LoopConfig{
		Model:          "gpt-4.1",
		Gateway:        sg,
		ContextManager: cm,
	}

	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "hello"}}, cfg)
	require.NoError(t, err)
	collectEvents(ch)

	assert.GreaterOrEqual(t, trimCallCount, 1, "ContextManager.Trim must be called at least once before LLM call (Fix V3)")
}

// TestRun_completionSignal_stopsLoop: when completion_signal is called, Run exits after next LLM response.
func TestRun_completionSignal_stopsLoop(t *testing.T) {
	callCount := 0
	gw := &countingGateway{
		responses: [][]Event{
			// First LLM call: fires completion_signal tool.
			{
				{Type: "tool_call", ToolCalls: []ToolCall{
					{ID: "sig1", Name: "completion_signal", Arguments: json.RawMessage(`{"summary":"done"}`)},
				}},
				{Type: "done"},
			},
			// Second LLM call: final acknowledgement after tool result.
			{
				{Type: "text_delta", Delta: "phase complete"},
				{Type: "done"},
			},
		},
		callCount: &callCount,
	}

	flag := &completionFlag{}
	completionTool := makeCompletionSignalTool(flag)

	cfg := LoopConfig{
		Model:   "gpt-4.1",
		Gateway: gw,
		Tools:   []Tool{completionTool},
	}

	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "finish phase"}}, cfg)
	require.NoError(t, err)

	events := collectEvents(ch)
	require.NotEmpty(t, events)
	assert.Equal(t, "done", events[len(events)-1].Type)

	// Loop must NOT make a third LLM call (flag.signaled stops it after second call).
	assert.LessOrEqual(t, callCount, 2, "Run must stop after completion_signal is processed, not loop indefinitely")
}

// ---- test helpers (in-package helpers for loop tests) ----

// countingGateway is a test double that returns different Event sequences on successive Call() invocations.
type countingGateway struct {
	responses [][]Event
	callCount *int
}

func (g *countingGateway) IsAvailable(model string) bool { return true }

func (g *countingGateway) Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
	idx := *g.callCount
	*g.callCount++
	if idx >= len(g.responses) {
		// Fallback: return a done event if no more scripted responses.
		ch := make(chan Event, 1)
		ch <- Event{Type: "done"}
		close(ch)
		return ch, nil
	}
	evs := g.responses[idx]
	ch := make(chan Event, len(evs))
	for _, ev := range evs {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

// blockingGateway blocks on Call() until the context is cancelled.
type blockingGateway struct{}

func (g *blockingGateway) IsAvailable(model string) bool { return true }

func (g *blockingGateway) Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
	ch := make(chan Event, 1)
	go func() {
		defer close(ch)
		<-ctx.Done()
		ch <- Event{Type: "error", Err: ctx.Err()}
	}()
	return ch, nil
}

// countingContextManager records Trim invocations.
type countingContextManager struct {
	count *int
}

func (cm *countingContextManager) Trim(messages []Message, model string, maxTokens int) ([]Message, error) {
	*cm.count++
	return messages, nil
}

// filterByType returns events whose Type matches the given string.
func filterByType(events []Event, eventType string) []Event {
	var out []Event
	for _, ev := range events {
		if ev.Type == eventType {
			out = append(out, ev)
		}
	}
	return out
}
```

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestRun_" -v
```
Expected: FAIL — `Run` is not implemented yet.

**Step 3: Write minimal implementation**

Add `Run()` to the existing `internal/agentloop/loop.go`:

```go
// Run executes one full phase-turn: trim context → LLM call → drain events → execute tools → loop.
// Run is stateless: it does not know about phases, gates, or session state.
// It exits when:
//   a) the LLM emits "done" and no tool calls were accumulated in that turn, OR
//   b) the LLM emits "done" and the completion_signal tool was called (flag set by closure), OR
//   c) ctx is cancelled (emits error event, closes channel).
//
// Design notes:
//   - Fix V3: cfg.ContextManager.Trim() called before every LLM call if non-nil.
//   - Fix Y1: "tool_end" events carry ToolID = result.ID (matches original ToolCall.ID).
//   - Fix X2: "tool_call" events carry ToolCalls slice (all parallel calls from one assistant message).
//   - Gateway field must be set in cfg; Run panics if cfg.Gateway is nil.
//   - Tool results are appended to msgs as "tool_result" Messages for the next LLM call.
//   - The completion_signal tool is expected to be included in cfg.Tools by BuildLoopConfig (Fix N6).
func Run(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
	if cfg.Gateway == nil {
		return nil, fmt.Errorf("Run: cfg.Gateway must not be nil")
	}

	out := make(chan Event, 64)

	go func() {
		defer close(out)

		// Local flag for completion_signal detection.
		// Run() finds completion_signal in cfg.Tools (added by BuildLoopConfig).
		// We detect it by checking results after executeCalls.
		var completionSignalFired bool

		currentMsgs := make([]Message, len(msgs))
		copy(currentMsgs, msgs)

		for {
			// Fix V3: trim context before every LLM call.
			if cfg.ContextManager != nil {
				trimmed, err := cfg.ContextManager.Trim(currentMsgs, cfg.Model, cfg.MaxTokens)
				if err != nil {
					out <- Event{Type: "error", Err: fmt.Errorf("context trim: %w", err)}
					return
				}
				currentMsgs = trimmed
			}

			// Check context before calling gateway.
			select {
			case <-ctx.Done():
				out <- Event{Type: "error", Err: ctx.Err()}
				return
			default:
			}

			// Call the LLM via gateway.
			llmCh, err := cfg.Gateway.Call(ctx, currentMsgs, cfg)
			if err != nil {
				out <- Event{Type: "error", Err: fmt.Errorf("gateway call: %w", err)}
				return
			}

			// Drain LLM events; accumulate tool calls from this assistant turn.
			var assistantText string
			var pendingCalls []ToolCall

			for ev := range llmCh {
				switch ev.Type {
				case "text_delta":
					assistantText += ev.Delta
					out <- ev
				case "tool_call":
					// Fix X2: event carries all parallel calls from one assistant message.
					pendingCalls = append(pendingCalls, ev.ToolCalls...)
					out <- ev
				case "done":
					// "done" marks end of this assistant turn — do NOT forward yet.
					// We forward our own "done" at the very end of Run().
				case "error":
					out <- ev
					return
				default:
					// Forward unknown event types (future-proofing).
					out <- ev
				}
			}

			// Check context after draining LLM response.
			select {
			case <-ctx.Done():
				out <- Event{Type: "error", Err: ctx.Err()}
				return
			default:
			}

			// Build assistant message for conversation history.
			assistantMsg := Message{
				Role:      "assistant",
				Content:   assistantText,
				ToolCalls: pendingCalls,
			}
			currentMsgs = append(currentMsgs, assistantMsg)

			// If no tool calls, this turn is complete. Emit done and exit.
			if len(pendingCalls) == 0 {
				out <- Event{Type: "done"}
				return
			}

			// Execute all tool calls in parallel.
			results := executeCalls(ctx, pendingCalls, cfg.Tools, cfg)

			// Emit tool_end events and build tool_result messages.
			for _, result := range results {
				// Fix Y1: ToolID = result.ID (matches original ToolCall.ID).
				out <- Event{
					Type:       "tool_end",
					ToolID:     result.ID,
					ToolName:   result.Name,
					ToolResult: result.Output,
					ToolErr:    result.Err,
				}

				// Check if this was the completion_signal tool.
				if result.Name == "completion_signal" && result.Err == nil {
					completionSignalFired = true
				}

				// Fix Z1: propagate tool error into message content so LLM can recover.
				content := result.Output
				if content == "" && result.Err != nil {
					content = fmt.Sprintf("Error: %v", result.Err)
				}
				currentMsgs = append(currentMsgs, Message{
					Role:       "tool_result",
					Content:    content,
					ToolCallID: result.ID, // Fix Y1: correlates to ToolCall.ID
				})
			}

			// If completion_signal fired, one more LLM call for the agent to
			// acknowledge, then we exit (Harness will read the flag after event drain).
			// We detect this by making the next LLM call and exiting after it.
			if completionSignalFired {
				// Make one final LLM call so the agent can acknowledge, then done.
				if cfg.ContextManager != nil {
					trimmed, err := cfg.ContextManager.Trim(currentMsgs, cfg.Model, cfg.MaxTokens)
					if err != nil {
						out <- Event{Type: "error", Err: fmt.Errorf("context trim (final): %w", err)}
						return
					}
					currentMsgs = trimmed
				}

				finalCh, err := cfg.Gateway.Call(ctx, currentMsgs, cfg)
				if err != nil {
					out <- Event{Type: "error", Err: fmt.Errorf("final gateway call: %w", err)}
					return
				}
				for ev := range finalCh {
					if ev.Type == "done" {
						break
					}
					if ev.Type == "error" {
						out <- ev
						return
					}
					out <- ev
				}
				out <- Event{Type: "done"}
				return
			}

			// Loop: another LLM call with tool results appended.
		}
	}()

	return out, nil
}
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestRun_" -race -v
```
Expected: PASS — all 6 Run tests green with race detector.

Then run the full suite:
```
go test ./internal/agentloop/... -race -count=1
```
Expected: ALL tests pass.

**Step 5: Commit**

```
git add internal/agentloop/loop.go internal/agentloop/loop_run_test.go
git commit -m "feat(agentloop): Task 8 — Loop.Run() with context trim, parallel tool execution, tool_end Y1 fix, completion_signal detection"
```

---

### Task 9: GateEngine

**Files:**
- Create: `internal/agentloop/gate.go`
- Create: `internal/agentloop/gate_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/gate_test.go`:

```go
package agentloop

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sdp_dev/internal/harness"
)

// ---- helpers ----

// minimalContract returns a TaskContract with all gates disabled (any snapshot passes).
func minimalContract() *harness.TaskContract {
	return &harness.TaskContract{
		Version: "1.0",
		QualityGates: harness.QualityGates{
			Build:     false,
			Test:      false,
			Lint:      false,
			Typecheck: false,
		},
		// No required evidence, no required metrics, no acceptance criteria.
	}
}

// blockingContract returns a TaskContract requiring evidence that will never be in the snapshot.
func blockingContract() *harness.TaskContract {
	return &harness.TaskContract{
		Version:          "1.0",
		RequiredEvidence: []string{"proof_of_work"},
	}
}

func emptySnapshot(phase Role) PhaseSnapshot {
	return PhaseSnapshot{
		Phase:    phase,
		Evidence: nil,
		Quality:  make(map[string]bool),
	}
}

// ---- tests ----

// TestGateEngine_pass: compliance passes → GateResult.Escalated = false.
func TestGateEngine_pass(t *testing.T) {
	engine := NewGateEngine(minimalContract(), 5*time.Second)

	snap := emptySnapshot(RoleDiscover)
	result := engine.Evaluate(context.Background(), snap)

	assert.False(t, result.Escalated, "gate must not escalate when compliance passes")
	assert.False(t, result.Report.Blocked, "report must not be blocked when compliance passes")
}

// TestGateEngine_blocked: compliance is blocked → GateResult.Escalated = true.
func TestGateEngine_blocked(t *testing.T) {
	// blockingContract requires "proof_of_work" evidence — emptySnapshot has none.
	engine := NewGateEngine(blockingContract(), 5*time.Second)

	snap := emptySnapshot(RoleDiscover)
	result := engine.Evaluate(context.Background(), snap)

	assert.True(t, result.Report.Blocked, "report must be blocked when required evidence is missing")
	assert.True(t, result.Escalated, "blocked report must set Escalated=true (requires human decision)")
}

// TestGateEngine_timeout: evalFn takes longer than timeout → GateWarn violation, Escalated=true.
// Fix R2-3: timeout is NOT automatic pass — it triggers escalation.
func TestGateEngine_timeout(t *testing.T) {
	// Use a very short timeout to trigger it reliably in tests.
	engine := NewGateEngine(minimalContract(), 10*time.Millisecond)

	// Override the evalFn with one that blocks longer than the timeout.
	// We test via a custom engine variant with injectable eval function.
	var called atomic.Bool
	engine.evalFn = func(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport {
		called.Store(true)
		time.Sleep(200 * time.Millisecond) // longer than 10ms timeout
		return harness.ComplianceReport{Blocked: false}
	}

	snap := emptySnapshot(RoleDiscover)
	result := engine.Evaluate(context.Background(), snap)

	// Timeout must escalate, NOT silently pass.
	assert.True(t, result.Escalated,
		"Fix R2-3: gate timeout must escalate (require human decision), not auto-pass")
	assert.False(t, result.Report.Blocked,
		"timeout sets Escalated without Blocked — human reviews, automation doesn't block")

	// Must have a GateWarn violation explaining the timeout.
	require.NotEmpty(t, result.Report.GateResults, "timeout result must contain gate results")
	found := false
	for _, gr := range result.Report.GateResults {
		if gr.Status == harness.GateWarn {
			for _, v := range gr.Violations {
				if v.Type == harness.DriftProcessIncomplete {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "timeout must produce a GateWarn violation with DriftProcessIncomplete type")
}

// TestGateEngine_nilContract: nil contract → Blocked=true (harness.EvaluateCompliance contract guard).
func TestGateEngine_nilContract(t *testing.T) {
	engine := NewGateEngine(nil, 5*time.Second)

	snap := emptySnapshot(RoleDiscover)
	result := engine.Evaluate(context.Background(), snap)

	// harness.EvaluateCompliance returns Blocked=true when contract is nil.
	assert.True(t, result.Report.Blocked)
	assert.True(t, result.Escalated)
}

// TestGateEngine_defaultTimeout: NewGateEngine with zero timeout uses 5s default.
func TestGateEngine_defaultTimeout(t *testing.T) {
	engine := NewGateEngine(minimalContract(), 0)
	assert.Equal(t, 5*time.Second, engine.timeout, "zero timeout must default to 5s")
}

// TestGateEngine_contextAlreadyCancelled: cancelled context returns escalated result immediately.
func TestGateEngine_contextAlreadyCancelled(t *testing.T) {
	engine := NewGateEngine(minimalContract(), 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling Evaluate

	snap := emptySnapshot(RoleDiscover)
	// Must not hang; returns escalated result.
	done := make(chan GateResult, 1)
	go func() {
		done <- engine.Evaluate(ctx, snap)
	}()

	select {
	case result := <-done:
		// Either the ctx cancellation was noticed immediately or EvaluateCompliance finished first.
		// Either way, function must return promptly.
		_ = result
	case <-time.After(2 * time.Second):
		t.Fatal("Evaluate blocked too long with cancelled context")
	}
}
```

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestGateEngine" -v
```
Expected: FAIL — `GateEngine`, `NewGateEngine` undefined.

**Step 3: Write minimal implementation**

Create `internal/agentloop/gate.go`:

```go
package agentloop

import (
	"context"
	"time"

	"sdp_dev/internal/harness"
)

// GateEngine wraps harness.EvaluateCompliance with a circuit breaker timeout.
//
// Critical note: harness.EvaluateCompliance does NOT accept a context parameter.
// Signature: EvaluateCompliance(contract *TaskContract, snapshot *TaskSnapshot) ComplianceReport
// GateEngine wraps it in a goroutine and uses select on result channel + evalCtx.Done().
// Fix N4: goroutine selects on both result channel and evalCtx.Done() — no hang after timeout.
// Fix R2-3: timeout → Escalated=true with GateWarn, NOT silent pass.
type GateEngine struct {
	contract *harness.TaskContract
	timeout  time.Duration

	// evalFn is the evaluation function. Defaults to harness.EvaluateCompliance.
	// Overridable in tests to inject slow/blocking behavior for timeout tests.
	evalFn func(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport
}

// NewGateEngine creates a GateEngine with the given contract and timeout.
// If timeout is zero, defaults to 5 seconds.
func NewGateEngine(contract *harness.TaskContract, timeout time.Duration) *GateEngine {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &GateEngine{
		contract: contract,
		timeout:  timeout,
		evalFn:   harness.EvaluateCompliance,
	}
}

// Evaluate runs compliance evaluation with a circuit breaker timeout.
// If evaluation completes in time: returns GateResult with Escalated=true iff report.Blocked.
// If evaluation times out: returns GateResult with Escalated=true + GateWarn violation (Fix R2-3).
// If context is already cancelled: treated as timeout (also escalates).
func (g *GateEngine) Evaluate(ctx context.Context, snap PhaseSnapshot) GateResult {
	evalCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	ch := make(chan harness.ComplianceReport, 1)

	// Fix N4: goroutine selects on both ch and evalCtx.Done() — exits promptly on timeout.
	go func() {
		report := g.evalFn(g.contract, snap.toHarness())
		select {
		case ch <- report:
		case <-evalCtx.Done():
			// Timeout already fired while we were evaluating — just discard result.
		}
	}()

	select {
	case report := <-ch:
		if report.Blocked {
			// Gate blocked → escalate for human decision.
			return GateResult{Report: report, Escalated: true}
		}
		return GateResult{Report: report, Escalated: false}

	case <-evalCtx.Done():
		// Fix R2-3: timeout is NOT automatic pass. Escalated=true requires human review.
		// Blocked=false so the automated path does not block; Escalated triggers human gate.
		return GateResult{
			Report: harness.ComplianceReport{
				Blocked: false,
				GateResults: []harness.GateResult{{
					GateID: "gate_timeout",
					Status: harness.GateWarn,
					Violations: []harness.Violation{{
						Type:    harness.DriftProcessIncomplete,
						Message: "gate evaluation timed out — human review required before transition",
					}},
				}},
			},
			Escalated: true,
		}
	}
}
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestGateEngine" -race -v
```
Expected: PASS — all 6 gate tests green.

Run the full suite:
```
go test ./internal/agentloop/... -race -count=1
```
Expected: all tests pass.

**Step 5: Commit**

```
git add internal/agentloop/gate.go internal/agentloop/gate_test.go
git commit -m "feat(agentloop): Task 9 — GateEngine with 5s circuit breaker, timeout escalation (Fix R2-3, N4); EvaluateCompliance wrapped without context"
```

---

### Task 10: PhaseRouter + ToolRegistry

**Files:**
- Create: `internal/agentloop/router.go`
- Create: `internal/agentloop/tools.go`
- Create: `internal/agentloop/router_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/router_test.go`:

```go
package agentloop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- helpers ----

// stubTool creates a minimal Tool with the given name for registry population.
func stubTool(name string) Tool {
	return Tool{
		Name:        name,
		Description: name + " stub",
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			return "stub:" + name, nil
		},
	}
}

// buildTestRegistry creates a ToolRegistry with all tools needed by DefaultPhaseMap.
func buildTestRegistry() *ToolRegistry {
	return NewToolRegistry([]Tool{
		stubTool("web_search"),
		stubTool("read_file"),
		stubTool("bd_search"),
		stubTool("glob"),
		stubTool("bd_create"),
		stubTool("edit_file"),
		stubTool("bash"),
		stubTool("grep"),
		stubTool("bd_comment"),
		// completion_signal is intentionally NOT in the registry (Fix N6).
	})
}

// ---- PhaseRouter: ResolveModel tests ----

// TestPhaseRouter_resolveModel_picksFirst: first model in list is returned when available.
func TestPhaseRouter_resolveModel_picksFirst(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})
	sg.AddResponse("openai/gpt-4.1", []Event{{Type: "done"}})

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)

	model, err := router.ResolveModel(RoleDiscover)
	require.NoError(t, err)
	assert.Equal(t, "deepseek/deepseek-v3.2", model,
		"first available model must be selected (deepseek is first in DefaultPhaseMap for discover)")
}

// TestPhaseRouter_resolveModel_fallsBack: first model unavailable → second model used.
func TestPhaseRouter_resolveModel_fallsBack(t *testing.T) {
	sg := NewStubGateway()
	// Only register the second model for discover phase.
	sg.AddResponse("openai/gpt-4.1", []Event{{Type: "done"}})
	// deepseek is NOT registered → IsAvailable returns false for it.

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)

	model, err := router.ResolveModel(RoleDiscover)
	require.NoError(t, err)
	assert.Equal(t, "openai/gpt-4.1", model,
		"must fall back to second model when first is unavailable")
}

// TestPhaseRouter_resolveModel_noneAvailable: no models available → error returned.
func TestPhaseRouter_resolveModel_noneAvailable(t *testing.T) {
	sg := NewStubGateway()
	// No models registered → all IsAvailable calls return false.

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)

	_, err := router.ResolveModel(RoleDiscover)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available model")
}

// ---- PhaseRouter: BuildLoopConfig tests ----

// TestPhaseRouter_buildLoopConfig_injectsCompletionSignal: completion_signal always present (Fix N6).
func TestPhaseRouter_buildLoopConfig_injectsCompletionSignal(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)
	acc := NewEvidenceAccumulator()
	flag := &completionFlag{}

	cfg, err := router.BuildLoopConfig(RoleDiscover, acc, flag, nil)
	require.NoError(t, err)

	toolNames := make([]string, len(cfg.Tools))
	for i, t := range cfg.Tools {
		toolNames[i] = t.Name
	}
	assert.Contains(t, toolNames, "completion_signal",
		"Fix N6: BuildLoopConfig must inject completion_signal — it must not be in ToolRegistry")
}

// TestPhaseRouter_buildLoopConfig_completionSignalNotInRegistry: Fix N6 — registry never has it.
func TestPhaseRouter_buildLoopConfig_completionSignalNotInRegistry(t *testing.T) {
	registry := buildTestRegistry()
	// Directly check registry: completion_signal must not be present.
	discoverCfg := DefaultPhaseMap[RoleDiscover]
	phaseTools := registry.ForPhase(discoverCfg)

	for _, tool := range phaseTools {
		assert.NotEqual(t, "completion_signal", tool.Name,
			"Fix N6: ToolRegistry must never return completion_signal — it is injected by BuildLoopConfig")
	}
}

// TestPhaseRouter_buildLoopConfig_wiresAfterToolCall: AfterToolCall is acc.OnToolResult.
func TestPhaseRouter_buildLoopConfig_wiresAfterToolCall(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)
	acc := NewEvidenceAccumulator()
	flag := &completionFlag{}

	cfg, err := router.BuildLoopConfig(RoleDiscover, acc, flag, nil)
	require.NoError(t, err)

	// AfterToolCall must be non-nil (wired to acc.OnToolResult).
	require.NotNil(t, cfg.AfterToolCall, "AfterToolCall must be wired to EvidenceAccumulator.OnToolResult")

	// Call it and verify evidence accumulates.
	err = cfg.AfterToolCall(ToolResult{ID: "tc1", Name: "bash", Output: "PASS: TestFoo"})
	require.NoError(t, err)
	snap := acc.Snapshot(RoleDiscover)
	assert.True(t, snap.Quality["test"], "AfterToolCall must route to EvidenceAccumulator")
}

// TestPhaseRouter_buildLoopConfig_wiresContextManager: ContextManager from constructor is wired (Fix W2).
func TestPhaseRouter_buildLoopConfig_wiresContextManager(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})

	trimCallCount := 0
	cm := &countingContextManager{count: &trimCallCount}

	// Fix X1: NewPhaseRouter takes cm as explicit parameter.
	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, cm)
	acc := NewEvidenceAccumulator()
	flag := &completionFlag{}

	cfg, err := router.BuildLoopConfig(RoleDiscover, acc, flag, nil)
	require.NoError(t, err)

	assert.NotNil(t, cfg.ContextManager, "Fix W2: ContextManager must be wired from NewPhaseRouter parameter")
	// Verify it's our cm by calling Trim.
	_, _ = cfg.ContextManager.Trim(nil, "model", 0)
	assert.Equal(t, 1, trimCallCount, "ContextManager must be the one passed to NewPhaseRouter")
}

// TestPhaseRouter_buildLoopConfig_nilContextManager: nil cm → cfg.ContextManager is nil (passthrough).
func TestPhaseRouter_buildLoopConfig_nilContextManager(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)
	acc := NewEvidenceAccumulator()
	flag := &completionFlag{}

	cfg, err := router.BuildLoopConfig(RoleDiscover, acc, flag, nil)
	require.NoError(t, err)
	assert.Nil(t, cfg.ContextManager, "nil ContextManager must produce nil in LoopConfig (passthrough for MVP)")
}

// TestPhaseRouter_buildLoopConfig_beforeToolCallWired: before func passed through to LoopConfig.
func TestPhaseRouter_buildLoopConfig_beforeToolCallWired(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)
	acc := NewEvidenceAccumulator()
	flag := &completionFlag{}

	called := false
	before := func(name string, args json.RawMessage) error {
		called = true
		return nil
	}

	cfg, err := router.BuildLoopConfig(RoleDiscover, acc, flag, before)
	require.NoError(t, err)
	require.NotNil(t, cfg.BeforeToolCall)

	// Invoke it to verify it's our function.
	_ = cfg.BeforeToolCall("bash", nil)
	assert.True(t, called)
}

// ---- PhaseRouter: NextPhase tests ----

// TestPhaseRouter_nextPhase: discover → plan (first AllowedNext).
func TestPhaseRouter_nextPhase(t *testing.T) {
	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), NewStubGateway(), nil)

	next := router.NextPhase(RoleDiscover)
	assert.Equal(t, RolePlan, next, "discover → plan (AllowedNext[0])")
}

// TestPhaseRouter_nextPhase_planToBuild: plan → build.
func TestPhaseRouter_nextPhase_planToBuild(t *testing.T) {
	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), NewStubGateway(), nil)

	next := router.NextPhase(RolePlan)
	assert.Equal(t, RoleBuild, next)
}

// TestPhaseRouter_nextPhase_finalPhase: eval has empty AllowedNext → returns eval (no transition).
func TestPhaseRouter_nextPhase_finalPhase(t *testing.T) {
	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), NewStubGateway(), nil)

	next := router.NextPhase(RoleEval)
	assert.Equal(t, RoleEval, next,
		"final phase with empty AllowedNext must return current phase (no transition)")
}

// ---- PhaseRouter: RecoveryPhase tests ----

// TestPhaseRouter_recoveryPhase: returns first from RecoveryNext.
func TestPhaseRouter_recoveryPhase(t *testing.T) {
	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), NewStubGateway(), nil)

	// build RecoveryNext = [plan, build] → returns plan.
	recovery := router.RecoveryPhase(RoleBuild)
	assert.Equal(t, RolePlan, recovery, "RecoveryPhase must return first entry in RecoveryNext")
}

// TestPhaseRouter_recoveryPhase_discoverStaysOnDiscover: discover RecoveryNext=[discover].
func TestPhaseRouter_recoveryPhase_discoverStaysOnDiscover(t *testing.T) {
	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), NewStubGateway(), nil)

	recovery := router.RecoveryPhase(RoleDiscover)
	assert.Equal(t, RoleDiscover, recovery)
}

// ---- ToolRegistry tests ----

// TestToolRegistry_forPhase_onlyAllowlisted: tools not in allowlist are excluded.
func TestToolRegistry_forPhase_onlyAllowlisted(t *testing.T) {
	registry := NewToolRegistry([]Tool{
		stubTool("web_search"),
		stubTool("read_file"),
		stubTool("edit_file"), // not in discover allowlist
		stubTool("bash"),      // not in discover allowlist
	})

	discoverCfg := DefaultPhaseMap[RoleDiscover]
	// DefaultPhaseMap[discover].Tools = ["web_search", "read_file", "bd_search"]
	// "edit_file" and "bash" are NOT in the allowlist.

	result := registry.ForPhase(discoverCfg)
	names := make([]string, len(result))
	for i, t := range result {
		names[i] = t.Name
	}

	assert.Contains(t, names, "web_search")
	assert.Contains(t, names, "read_file")
	assert.NotContains(t, names, "edit_file",
		"edit_file is not in discover allowlist — must be excluded")
	assert.NotContains(t, names, "bash",
		"bash is not in discover allowlist — must be excluded")
}

// TestToolRegistry_forPhase_missingToolSkipped: allowlisted name not in registry → silently omitted.
func TestToolRegistry_forPhase_missingToolSkipped(t *testing.T) {
	// Registry only has web_search; discover allowlist also needs read_file and bd_search.
	registry := NewToolRegistry([]Tool{stubTool("web_search")})

	discoverCfg := DefaultPhaseMap[RoleDiscover]
	result := registry.ForPhase(discoverCfg)

	// Only web_search is returned; missing tools are silently omitted (not an error).
	require.Len(t, result, 1)
	assert.Equal(t, "web_search", result[0].Name)
}

// TestToolRegistry_forPhase_completionSignalNeverReturned: even if accidentally registered, excluded (Fix N6).
func TestToolRegistry_forPhase_completionSignalNeverReturned(t *testing.T) {
	// This tests defensive behavior: even if someone registers completion_signal, it
	// should not appear in ForPhase results because it's not in any PhaseConfig.Tools allowlist.
	registry := NewToolRegistry([]Tool{
		stubTool("web_search"),
		stubTool("completion_signal"), // accidentally registered
	})

	discoverCfg := DefaultPhaseMap[RoleDiscover]
	result := registry.ForPhase(discoverCfg)

	for _, tool := range result {
		assert.NotEqual(t, "completion_signal", tool.Name,
			"Fix N6: completion_signal must not be in any phase allowlist")
	}
}

// TestDefaultPhaseMap_noCompletionSignalInAllowlists: Fix N6 — structural guarantee.
func TestDefaultPhaseMap_noCompletionSignalInAllowlists(t *testing.T) {
	for phase, cfg := range DefaultPhaseMap {
		for _, toolName := range cfg.Tools {
			assert.NotEqual(t, "completion_signal", toolName,
				"Fix N6: completion_signal must not appear in any PhaseConfig.Tools allowlist (phase: %s)", phase)
		}
	}
}

// TestNewPhaseRouter_gatewayWired: gateway is accessible via ResolveModel.
func TestNewPhaseRouter_gatewayWired(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("openai/gpt-4.1", []Event{{Type: "done"}})

	// plan phase: ["openai/gpt-4.1", "anthropic/claude-opus-4-5"]
	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)

	model, err := router.ResolveModel(RolePlan)
	require.NoError(t, err)
	assert.Equal(t, "openai/gpt-4.1", model)
}
```

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestPhaseRouter|TestToolRegistry|TestDefaultPhaseMap" -v
```
Expected: FAIL — `PhaseRouter`, `NewPhaseRouter`, `DefaultPhaseMap`, `ToolRegistry`, `NewToolRegistry` undefined.

**Step 3: Write minimal implementation**

Create `internal/agentloop/tools.go`:

```go
package agentloop

// ToolRegistry holds all available tools and filters them per phase allowlist.
// Static for MVP — dynamic registration is backlog (I-tool-registry).
// Fix N6: ToolRegistry never contains "completion_signal".
//   BuildLoopConfig injects it explicitly with the phase-specific completionFlag closure.
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a ToolRegistry from the given tool slice.
// Duplicate names are silently last-wins (later entry overwrites earlier).
func NewToolRegistry(tools []Tool) *ToolRegistry {
	m := make(map[string]Tool, len(tools))
	for _, t := range tools {
		m[t.Name] = t
	}
	return &ToolRegistry{tools: m}
}

// ForPhase returns only the tools whose names appear in cfg.Tools (the phase allowlist).
// Loop receives this already-filtered slice — it cannot call tools outside the allowlist.
// Missing tools (in allowlist but not in registry) are silently omitted.
func (tr *ToolRegistry) ForPhase(cfg PhaseConfig) []Tool {
	result := make([]Tool, 0, len(cfg.Tools))
	for _, name := range cfg.Tools {
		if t, ok := tr.tools[name]; ok {
			result = append(result, t)
		}
	}
	return result
}
```

Create `internal/agentloop/router.go`:

```go
package agentloop

import (
	"encoding/json"
	"fmt"
)

// DefaultPhaseMap is the standard SDP phase configuration.
// Fix N6: completion_signal is intentionally absent from ALL Tools allowlists.
//   BuildLoopConfig adds it implicitly — ToolRegistry must never contain it.
var DefaultPhaseMap = map[Role]PhaseConfig{
	RoleDiscover: {
		Models:          []string{"deepseek/deepseek-v3.2", "openai/gpt-4.1"},
		Tools:           []string{"web_search", "read_file", "bd_search"}, // no completion_signal (N6)
		AllowedNext:     []Role{RolePlan},
		RecoveryNext:    []Role{RoleDiscover},
		GateRequired:    true,
		MinOutputTokens: 200,
	},
	RolePlan: {
		Models:       []string{"openai/gpt-4.1", "anthropic/claude-opus-4-5"},
		Tools:        []string{"read_file", "glob", "bd_create"}, // no completion_signal (N6)
		AllowedNext:  []Role{RoleBuild},
		RecoveryNext: []Role{RoleDiscover, RolePlan},
		GateRequired: true,
	},
	RoleBuild: {
		Models:       []string{"anthropic/claude-sonnet-4-6", "openai/gpt-4.1"},
		Tools:        []string{"read_file", "edit_file", "bash", "glob"}, // no completion_signal (N6)
		AllowedNext:  []Role{RoleReview},
		RecoveryNext: []Role{RolePlan, RoleBuild},
		GateRequired: true,
	},
	RoleReview: {
		Models:       []string{"openai/gpt-4.1", "deepseek/deepseek-v3.2"},
		Tools:        []string{"read_file", "grep", "bd_comment"}, // no completion_signal (N6)
		AllowedNext:  []Role{RoleEval, RoleBuild},
		RecoveryNext: []Role{RoleBuild},
		GateRequired: true,
	},
	RoleEval: {
		Models:       []string{"anthropic/claude-sonnet-4-6", "openai/gpt-4.1"},
		Tools:        []string{"bash", "read_file"}, // no completion_signal (N6)
		AllowedNext:  []Role{},                      // final phase — no transition
		RecoveryNext: []Role{RoleBuild},
		GateRequired: true,
	},
}

// PhaseRouter maps phases to models, tools, and system prompts.
// Fix W2 (v11): contextManager field added; nil = passthrough (MVP).
// Fix X1 (v12): constructor accepts contextManager as explicit parameter.
type PhaseRouter struct {
	phaseMap       map[Role]PhaseConfig
	registry       *ToolRegistry
	gateway        ModelGateway
	contextManager ContextManager // wired into LoopConfig by BuildLoopConfig
}

// NewPhaseRouter creates a PhaseRouter.
// cm may be nil — nil means passthrough (no context trimming) for MVP sessions.
// Fix X1: cm is an explicit parameter so callers cannot accidentally leave it nil via field assignment.
func NewPhaseRouter(
	phaseMap map[Role]PhaseConfig,
	registry *ToolRegistry,
	gateway ModelGateway,
	cm ContextManager, // Fix X1: explicit, nil = passthrough
) *PhaseRouter {
	return &PhaseRouter{
		phaseMap:       phaseMap,
		registry:       registry,
		gateway:        gateway,
		contextManager: cm,
	}
}

// ResolveModel tries models from PhaseConfig.Models in priority order.
// Returns the first model for which gateway.IsAvailable() returns true.
// AgentContext.Model is set from this result — it does not change during a phase run (I2).
func (r *PhaseRouter) ResolveModel(phase Role) (string, error) {
	cfg, ok := r.phaseMap[phase]
	if !ok {
		return "", fmt.Errorf("no phase config for role %q", phase)
	}
	for _, m := range cfg.Models {
		if r.gateway.IsAvailable(m) {
			return m, nil
		}
	}
	return "", fmt.Errorf("no available model for phase %s (tried: %v)", phase, cfg.Models)
}

// NextPhase returns the happy-path next phase (first AllowedNext entry).
// If AllowedNext is empty (final phase), returns current (no transition).
// Fix R2-4: defined on PhaseRouter — Harness does not compute transitions itself.
func (r *PhaseRouter) NextPhase(current Role) Role {
	cfg := r.phaseMap[current]
	if len(cfg.AllowedNext) == 0 {
		return current // final phase — RoleEval
	}
	return cfg.AllowedNext[0]
}

// RecoveryPhase returns the rollback-path phase (first RecoveryNext entry).
// If RecoveryNext is empty, returns current (warn, not crash).
// Fix R2-4: defined on PhaseRouter — Harness does not compute transitions itself.
func (r *PhaseRouter) RecoveryPhase(current Role) Role {
	cfg := r.phaseMap[current]
	if len(cfg.RecoveryNext) == 0 {
		return current // no recovery path — stay, wait for override
	}
	return cfg.RecoveryNext[0]
}

// BuildLoopConfig assembles a LoopConfig for the given phase.
// Fix N6: appends completion_signal tool (created via makeCompletionSignalTool(flag)) AFTER
//   ForPhase() — it is NOT in ToolRegistry and NOT in PhaseConfig.Tools.
// Fix U2: before func is passed explicitly (nil = no-op; Harness wires h.beforeToolCall here).
// Fix W2: LoopConfig.ContextManager = r.contextManager (nil = passthrough for MVP).
// Fix R2-2: flag is an explicit *completionFlag; RunPhase reads it after draining events.
func (r *PhaseRouter) BuildLoopConfig(
	phase Role,
	acc *EvidenceAccumulator,
	flag *completionFlag,
	before func(name string, args json.RawMessage) error, // Fix U2: explicit, nil = no-op
) (LoopConfig, error) {
	model, err := r.ResolveModel(phase)
	if err != nil {
		return LoopConfig{}, err
	}

	cfg := r.phaseMap[phase]

	// Get allowlisted tools from registry (does not include completion_signal — Fix N6).
	tools := r.registry.ForPhase(cfg)

	// Fix N6: append completion_signal with captured flag pointer.
	// Registered exactly once per BuildLoopConfig call — no duplication.
	tools = append(tools, makeCompletionSignalTool(flag))

	return LoopConfig{
		Model:          model,
		SystemPrompt:   cfg.SystemPrompt,
		Tools:          tools,
		BeforeToolCall: before,           // Fix U2: wired explicitly
		AfterToolCall:  acc.OnToolResult, // evidence extraction from every tool result
		ContextManager: r.contextManager, // Fix W2: nil = passthrough
		Gateway:        r.gateway,        // wired so Run() can call LLM
	}, nil
}
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestPhaseRouter|TestToolRegistry|TestDefaultPhaseMap" -race -v
```
Expected: PASS — all router + registry tests green.

Run the full suite one final time:
```
go test ./internal/agentloop/... -race -count=1 -v 2>&1 | tail -10
```
Expected:
```
--- PASS: TestDefaultPhaseMap_noCompletionSignalInAllowlists (...)
--- PASS: TestToolRegistry_forPhase_completionSignalNeverReturned (...)
PASS
ok  	sdp_dev/internal/agentloop	...s
```

**Step 5: Commit**

```
git add internal/agentloop/router.go internal/agentloop/tools.go internal/agentloop/router_test.go
git commit -m "feat(agentloop): Task 10 — PhaseRouter (ResolveModel, NextPhase, RecoveryPhase, BuildLoopConfig), ToolRegistry.ForPhase, DefaultPhaseMap (Fix N6, W2, X1)"
```

---

## Final verification

After all five tasks (6–10) are committed, run the complete test suite:

```bash
go build ./internal/agentloop/...
go test ./internal/agentloop/... -race -count=1 -v
```

Expected: zero compilation errors, all tests pass (Tasks 1–10 combined), no race detector warnings.

---

## Notes for the implementer

1. **EvaluateCompliance has no context parameter.** The real signature is `EvaluateCompliance(contract *TaskContract, snapshot *TaskSnapshot) ComplianceReport`. The design spec shows a `ctx` parameter that does not exist in the implementation. GateEngine wraps it in a goroutine and uses `select` on the result channel and `evalCtx.Done()`.

2. **Tool.Execute context type fix.** The foundation plan (Task 1) used `interface{}` for the context parameter in `Tool.Execute` to avoid importing `context` in a types-only file. This must be corrected to `context.Context` with `import "context"` in `types.go` before Task 7 compiles.

3. **Gateway in LoopConfig.** The design spec's `Run()` signature shows `func Run(ctx, msgs, cfg LoopConfig)`. Gateway is added as a field `Gateway ModelGateway` to `LoopConfig` in Task 7 so that `Run()` can call the LLM without additional parameters. This is consistent with the spec discussion ("must be in cfg or passed separately — add Gateway ModelGateway to LoopConfig").

4. **completionFlag is package-private.** It is defined in `types.go` (Task 1) and used in both `loop.go` (makeCompletionSignalTool) and `harness.go` (RunPhase). Tests in the `agentloop` package (internal tests without `_test` suffix) can reference it directly.

5. **StubGateway returns the same sequence every Call.** For tests requiring different responses on successive calls, use `countingGateway` (defined in `loop_run_test.go`). This test helper is in the internal test package and not exported.

6. **GateEngine.evalFn is injectable for tests.** The `evalFn` field allows tests to inject slow/blocking evaluation functions to trigger the timeout circuit breaker without actually waiting 5 seconds. This pattern is acceptable for test-only field access since gate tests are internal package tests.

7. **AfterToolCall for rejected calls.** When `BeforeToolCall` rejects a call, `AfterToolCall` is still called with the rejection `ToolResult` (containing the hook error in `Err`). This is required so `EvidenceAccumulator` can record the failure as negative evidence. Both errors (hook rejection + callback error) are wrapped using `%w; callback: %v` to preserve both in the chain.

8. **LoopConfig.Gateway is set by BuildLoopConfig.** Callers should use `BuildLoopConfig` (which sets the gateway from `r.gateway`) rather than constructing `LoopConfig` manually. This ensures the gateway is always wired.

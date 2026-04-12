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

// TestExecuteCalls_afterCallbackError_onToolFailure: Fix R6 — when tool.Execute fails AND
// AfterToolCall also fails, BOTH errors must appear in ToolResult.Err (neither overwritten).
func TestExecuteCalls_afterCallbackError_onToolFailure(t *testing.T) {
	tools := []Tool{makeTool("bash", "", errors.New("exec failed"))}
	calls := []ToolCall{{ID: "tc1", Name: "bash", Arguments: json.RawMessage(`{}`)}}

	cfg := LoopConfig{
		AfterToolCall: func(r ToolResult) error {
			return fmt.Errorf("callback also failed")
		},
	}

	results := executeCalls(context.Background(), calls, tools, cfg)
	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	// Fix R6: original tool error must NOT be lost.
	errMsg := results[0].Err.Error()
	assert.Contains(t, errMsg, "exec failed", "Fix R6: tool.Execute error must be preserved")
	assert.Contains(t, errMsg, "callback", "Fix R6: callback error must also be present")
}

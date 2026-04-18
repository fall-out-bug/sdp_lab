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
	sg.AddResponse("glm-5", []Event{{Type: "done"}})
	sg.AddResponse("glm-4.7", []Event{{Type: "done"}})

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)

	model, err := router.ResolveModel(RoleDiscover)
	require.NoError(t, err)
	assert.Equal(t, "glm-5", model,
		"first available model must be selected (glm-5 is first in DefaultPhaseMap for discover)")
}

// TestPhaseRouter_resolveModel_fallsBack: first model unavailable → second model used.
func TestPhaseRouter_resolveModel_fallsBack(t *testing.T) {
	sg := NewStubGateway()
	// Only register the second model for discover phase.
	sg.AddResponse("glm-4.7", []Event{{Type: "done"}})
	// glm-5 is NOT registered → IsAvailable returns false for it.

	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)

	model, err := router.ResolveModel(RoleDiscover)
	require.NoError(t, err)
	assert.Equal(t, "glm-4.7", model,
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
	sg.AddResponse("glm-5", []Event{{Type: "done"}})

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
	sg.AddResponse("glm-5", []Event{{Type: "done"}})

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
	sg.AddResponse("glm-5", []Event{{Type: "done"}})

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
	sg.AddResponse("glm-5", []Event{{Type: "done"}})

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
	sg.AddResponse("glm-5", []Event{{Type: "done"}})

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
	sg.AddResponse("glm-4.7", []Event{{Type: "done"}})

	// plan phase: ["glm-5", "glm-4.7"]
	router := NewPhaseRouter(DefaultPhaseMap, buildTestRegistry(), sg, nil)

	model, err := router.ResolveModel(RolePlan)
	require.NoError(t, err)
	assert.Equal(t, "glm-4.7", model)
}

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
//
//	ForPhase() — it is NOT in ToolRegistry and NOT in PhaseConfig.Tools.
//
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

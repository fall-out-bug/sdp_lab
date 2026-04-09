// Package kernel provides core types and interfaces for AI agent runtime orchestration.
// It defines the data structures used for session management, artifact tracking,
// tool policy enforcement, and event tracing across the SDP system.
package kernel

import (
	"encoding/json"
	"time"
)

// RunID uniquely identifies a single runtime execution of an agent.
type RunID string

// SessionID uniquely identifies a conversation session across multiple runs.
type SessionID string

// ArtifactType represents the category of an artifact produced or consumed during execution.
type ArtifactType string

const (
	// ArtifactDispatchPacket represents the input parameters sent to an agent.
	ArtifactDispatchPacket ArtifactType = "dispatch_packet"
	// ArtifactResultPacket represents the output produced by an agent.
	ArtifactResultPacket ArtifactType = "result_packet"
	// ArtifactEvidence represents verification data produced during execution.
	ArtifactEvidence ArtifactType = "evidence"
	// ArtifactProvenance represents metadata tracking the origin and history of artifacts.
	ArtifactProvenance ArtifactType = "provenance"
	// ArtifactContract represents a policy or agreement document.
	ArtifactContract ArtifactType = "contract"
	// ArtifactIntake represents data ingested from external sources.
	ArtifactIntake ArtifactType = "intake"
)

// ArtifactRef references an artifact with metadata for tracking and verification.
type ArtifactRef struct {
	Type      ArtifactType `json:"type"`
	Path      string       `json:"path"`
	Hash      string       `json:"hash,omitempty"`
	CreatedAt time.Time    `json:"created_at,omitempty"`
	Size      int64        `json:"size,omitempty"`
}

// ContextSegment represents a piece of context (code, docs, etc.) included in a prompt.
type ContextSegment struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Source     string `json:"source,omitempty"`
	Content    string `json:"content,omitempty"`
	TokenCount int    `json:"token_count,omitempty"`
}

// CapabilitySet describes the capabilities and limits of an AI runtime.
type CapabilitySet struct {
	Vision            bool `json:"vision,omitempty"`
	ToolCalling       bool `json:"tool_calling,omitempty"`
	Streaming         bool `json:"streaming,omitempty"`
	ReasoningControls bool `json:"reasoning_controls,omitempty"`
	MaxContextTokens  int  `json:"max_context_tokens,omitempty"`
}

// AgentDefinition defines an AI agent including its role and required capabilities.
type AgentDefinition struct {
	ID                   string        `json:"id"`
	Role                 string        `json:"role"`
	Description          string        `json:"description,omitempty"`
	AllowedWorkflowPacks []string      `json:"allowed_workflow_packs,omitempty"`
	ToolPolicyRef        string        `json:"tool_policy_ref,omitempty"`
	RequiredCapabilities CapabilitySet `json:"required_capabilities,omitempty"`
}

// SessionState tracks the current state of an active agent session.
type SessionState struct {
	RunID            RunID             `json:"run_id"`
	SessionID        SessionID         `json:"session_id"`
	AgentID          string            `json:"agent_id,omitempty"`
	WorkflowPackRefs []string          `json:"workflow_pack_refs,omitempty"`
	ContextSegments  []ContextSegment  `json:"context_segments,omitempty"`
	ToolPolicyRef    string            `json:"tool_policy_ref,omitempty"`
	MemoryCandidates []MemoryCandidate `json:"memory_candidates,omitempty"`
	ArtifactRefs     []ArtifactRef     `json:"artifact_refs,omitempty"`
	TraceRefs        []string          `json:"trace_refs,omitempty"`
}

// WorkflowPack bundles together prompts, roles, and hooks for a specific workflow.
type WorkflowPack struct {
	ID              string             `json:"id"`
	Version         string             `json:"version"`
	Description     string             `json:"description,omitempty"`
	Dependencies    []string           `json:"dependencies,omitempty"`
	PromptFragments []PromptFragment   `json:"prompt_fragments,omitempty"`
	Roles           []RoleDefinition   `json:"roles,omitempty"`
	Hooks           []HookRegistration `json:"hooks,omitempty"`
	EvalRefs        []string           `json:"eval_refs,omitempty"`
}

// PromptFragment is a reusable piece of prompt content.
type PromptFragment struct {
	ID          string `json:"id"`
	Kind        string `json:"kind,omitempty"`
	Content     string `json:"content,omitempty"`
	Description string `json:"description,omitempty"`
}

// RoleDefinition defines a role within a workflow phase.
type RoleDefinition struct {
	ID                string   `json:"id"`
	Phase             string   `json:"phase"`
	Agent             string   `json:"agent,omitempty"`
	Description       string   `json:"description,omitempty"`
	PromptFragmentIDs []string `json:"prompt_fragment_ids,omitempty"`
}

// HookKind classifies the type of hook being registered.
type HookKind string

const (
	HookKindApproval        HookKind = "approval"
	HookKindToolPolicy      HookKind = "tool_policy"
	HookKindMemoryCandidate HookKind = "memory_candidate"
	HookKindTraceEnrichment HookKind = "trace_enrichment"
)

// HookRegistration represents a hook registered in the workflow system.
type HookRegistration struct {
	ID          string   `json:"id"`
	Kind        HookKind `json:"kind"`
	Description string   `json:"description,omitempty"`
}

type MemoryScope string

const (
	MemoryScopeUser    MemoryScope = "user"
	MemoryScopeProject MemoryScope = "project"
	MemoryScopeTask    MemoryScope = "task"
)

// MemoryCandidate represents a piece of memory proposed for storage.
type MemoryCandidate struct {
	ID         string      `json:"id"`
	Scope      MemoryScope `json:"scope"`
	Content    string      `json:"content,omitempty"`
	Confidence float64     `json:"confidence,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	TraceRefs  []string    `json:"trace_refs,omitempty"`
}

type ToolPolicyDecision string

const (
	ToolPolicyAllow ToolPolicyDecision = "allow"
	ToolPolicyAsk   ToolPolicyDecision = "ask"
	ToolPolicyDeny  ToolPolicyDecision = "deny"
)

// ToolPolicy defines rules for controlling tool usage.
type ToolPolicy struct {
	ID              string             `json:"id"`
	Description     string             `json:"description,omitempty"`
	Version         string             `json:"version,omitempty"`
	EvaluationShape []string           `json:"evaluation_shape,omitempty"`
	DefaultDecision ToolPolicyDecision `json:"default_decision,omitempty"`
}

// ToolCallRequest represents a request to call a tool.
type ToolCallRequest struct {
	Tool  string          `json:"tool"`
	Args  json.RawMessage `json:"args,omitempty"`
	Files []string        `json:"files,omitempty"`
}

// ToolCallDecision represents the decision made for a tool call request.
type ToolCallDecision struct {
	PolicyID string             `json:"policy_id,omitempty"`
	Decision ToolPolicyDecision `json:"decision"`
	Reason   string             `json:"reason,omitempty"`
}

// TraceEventKind classifies the type of trace event.
type TraceEventKind string

const (
	TraceEventSession  TraceEventKind = "session"
	TraceEventRouting  TraceEventKind = "routing"
	TraceEventTool     TraceEventKind = "tool"
	TraceEventMemory   TraceEventKind = "memory"
	TraceEventArtifact TraceEventKind = "artifact"
	TraceEventEval     TraceEventKind = "eval"
)

// TraceEvent represents an event in the execution trace.
type TraceEvent struct {
	ID            string          `json:"id,omitempty"`
	RunID         RunID           `json:"run_id,omitempty"`
	SessionID     SessionID       `json:"session_id,omitempty"`
	AgentID       string          `json:"agent_id,omitempty"`
	Kind          TraceEventKind  `json:"kind,omitempty"`
	Phase         string          `json:"phase,omitempty"`
	At            string          `json:"at,omitempty"`
	ParentID      string          `json:"parent_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Payload       json.RawMessage `json:"payload,omitempty"`
}

// ApprovalDecision represents the decision made for an approval request.
type ApprovalDecision string

const (
	ApprovalApprove  ApprovalDecision = "approve"
	ApprovalReject   ApprovalDecision = "reject"
	ApprovalEscalate ApprovalDecision = "escalate"
)

// ApprovalHook represents an approval request requiring human intervention.
type ApprovalHook struct {
	ID          string `json:"id"`
	RequestType string `json:"request_type"`
	Description string `json:"description,omitempty"`
}

// EvalCase represents a test case for evaluating workflow behavior.
type EvalCase struct {
	ID                       string               `json:"id"`
	Scenario                 string               `json:"scenario"`
	Inputs                   map[string]any       `json:"inputs,omitempty"`
	ExpectedTraceKinds       []TraceEventKind     `json:"expected_trace_kinds,omitempty"`
	ExpectedRoutingProviders []ProviderID         `json:"expected_routing_providers,omitempty"`
	ExpectedToolDecisions    []ToolPolicyDecision `json:"expected_tool_decisions,omitempty"`
	ExpectedArtifacts        []ArtifactType       `json:"expected_artifacts,omitempty"`
	ExpectedMemoryScopes     []MemoryScope        `json:"expected_memory_scopes,omitempty"`
}

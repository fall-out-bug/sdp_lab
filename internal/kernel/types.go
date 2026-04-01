package kernel

import (
	"encoding/json"
	"time"
)

type RunID string

type SessionID string

type ArtifactType string

const (
	ArtifactDispatchPacket ArtifactType = "dispatch_packet"
	ArtifactResultPacket   ArtifactType = "result_packet"
	ArtifactEvidence       ArtifactType = "evidence"
	ArtifactProvenance     ArtifactType = "provenance"
	ArtifactContract       ArtifactType = "contract"
	ArtifactIntake         ArtifactType = "intake"
)

type ArtifactRef struct {
	Type      ArtifactType `json:"type"`
	Path      string       `json:"path"`
	Hash      string       `json:"hash,omitempty"`
	CreatedAt time.Time    `json:"created_at,omitempty"`
	Size      int64        `json:"size,omitempty"`
}

type ContextSegment struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Source     string `json:"source,omitempty"`
	Content    string `json:"content,omitempty"`
	TokenCount int    `json:"token_count,omitempty"`
}

type CapabilitySet struct {
	Vision            bool `json:"vision,omitempty"`
	ToolCalling       bool `json:"tool_calling,omitempty"`
	Streaming         bool `json:"streaming,omitempty"`
	ReasoningControls bool `json:"reasoning_controls,omitempty"`
	MaxContextTokens  int  `json:"max_context_tokens,omitempty"`
}

type AgentDefinition struct {
	ID                   string        `json:"id"`
	Role                 string        `json:"role"`
	Description          string        `json:"description,omitempty"`
	AllowedWorkflowPacks []string      `json:"allowed_workflow_packs,omitempty"`
	ToolPolicyRef        string        `json:"tool_policy_ref,omitempty"`
	RequiredCapabilities CapabilitySet `json:"required_capabilities,omitempty"`
}

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

type PromptFragment struct {
	ID          string `json:"id"`
	Kind        string `json:"kind,omitempty"`
	Content     string `json:"content,omitempty"`
	Description string `json:"description,omitempty"`
}

type RoleDefinition struct {
	ID                string   `json:"id"`
	Phase             string   `json:"phase"`
	Agent             string   `json:"agent,omitempty"`
	Description       string   `json:"description,omitempty"`
	PromptFragmentIDs []string `json:"prompt_fragment_ids,omitempty"`
}

type HookKind string

const (
	HookKindApproval        HookKind = "approval"
	HookKindToolPolicy      HookKind = "tool_policy"
	HookKindMemoryCandidate HookKind = "memory_candidate"
	HookKindTraceEnrichment HookKind = "trace_enrichment"
)

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

type ToolPolicy struct {
	ID              string             `json:"id"`
	Description     string             `json:"description,omitempty"`
	Version         string             `json:"version,omitempty"`
	EvaluationShape []string           `json:"evaluation_shape,omitempty"`
	DefaultDecision ToolPolicyDecision `json:"default_decision,omitempty"`
}

type ToolCallRequest struct {
	Tool  string          `json:"tool"`
	Args  json.RawMessage `json:"args,omitempty"`
	Files []string        `json:"files,omitempty"`
}

type ToolCallDecision struct {
	PolicyID string             `json:"policy_id,omitempty"`
	Decision ToolPolicyDecision `json:"decision"`
	Reason   string             `json:"reason,omitempty"`
}

type TraceEventKind string

const (
	TraceEventSession  TraceEventKind = "session"
	TraceEventTool     TraceEventKind = "tool"
	TraceEventMemory   TraceEventKind = "memory"
	TraceEventArtifact TraceEventKind = "artifact"
	TraceEventEval     TraceEventKind = "eval"
)

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

type ApprovalDecision string

const (
	ApprovalApprove  ApprovalDecision = "approve"
	ApprovalReject   ApprovalDecision = "reject"
	ApprovalEscalate ApprovalDecision = "escalate"
)

type ApprovalHook struct {
	ID          string `json:"id"`
	RequestType string `json:"request_type"`
	Description string `json:"description,omitempty"`
}

type EvalCase struct {
	ID                    string               `json:"id"`
	Scenario              string               `json:"scenario"`
	Inputs                map[string]any       `json:"inputs,omitempty"`
	ExpectedTraceKinds    []TraceEventKind     `json:"expected_trace_kinds,omitempty"`
	ExpectedToolDecisions []ToolPolicyDecision `json:"expected_tool_decisions,omitempty"`
	ExpectedArtifacts     []ArtifactType       `json:"expected_artifacts,omitempty"`
	ExpectedMemoryScopes  []MemoryScope        `json:"expected_memory_scopes,omitempty"`
}

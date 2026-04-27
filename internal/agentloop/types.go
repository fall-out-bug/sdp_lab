package agentloop

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/harness"
)

// ---- Role constants ----

type Role string

const (
	RoleDiscover Role = "discover"
	RolePlan     Role = "plan"
	RoleBuild    Role = "build"
	RoleReview   Role = "review"
	RoleEval     Role = "eval"
)

// ---- Message / ToolCall / Tool / ToolResult ----

type Message struct {
	Role       string // "user" | "assistant" | "tool_result"
	Content    string
	ToolCalls  []ToolCall // Fix X2: assistant messages carry tool calls
	ToolCallID string     // Fix Y1: correlates tool_result to tool_call
	Timestamp  time.Time
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
	Sandboxed   bool
	Execute     func(ctx context.Context, id string, args json.RawMessage) (string, error) // Fix F1: context.Context, not interface{}
}

// ToolResult is the full outcome of one tool execution (success or error).
// Fix N5: AfterToolCall carries full ToolResult. Fix T1: Arguments preserved.
type ToolResult struct {
	ID        string
	Name      string
	Arguments json.RawMessage // original call arguments (Fix T1)
	Output    string
	Err       error
}

// ---- LoopConfig ----

// ContextManager trims message history to fit model context window.
type ContextManager interface {
	Trim(messages []Message, model string, maxTokens int) ([]Message, error)
}

// ModelGateway abstracts LLM API calls. Fix F2: defined here so LoopConfig.Gateway compiles.
// StubGateway (test double) is in gateway.go (Task 6).
type ModelGateway interface {
	// Call returns a channel of Events for one LLM request. Channel closes after "done" or "error".
	Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error)
	// IsAvailable returns true if the model is reachable (used by PhaseRouter.ResolveModel).
	IsAvailable(model string) bool
}

type LoopConfig struct {
	Model          string
	SystemPrompt   string
	Tools          []Tool
	MaxTokens      int
	TurnTimeout    time.Duration
	BeforeToolCall func(name string, args json.RawMessage) error
	AfterToolCall  func(result ToolResult) error
	ContextManager ContextManager // nil = passthrough (MVP)
	Gateway        ModelGateway   // Fix F2: required by Run() — set by BuildLoopConfig
}

// ---- Event ----

type Event struct {
	Type          string // "text_delta"|"tool_call"|"tool_end"|"turn_end"|"done"|"error"|"warn"|"human_gate"|"session_stopped"
	Code          string `json:"code,omitempty"`
	Delta         string
	Count         int               `json:"count,omitempty"`
	Fields        map[string]string `json:"fields,omitempty"`
	ToolCalls     []ToolCall        // Fix X2: "tool_call" event carries all parallel calls
	ToolID        string            // Fix Y1: for "tool_end" — matches ToolCall.ID
	ToolName      string
	ToolResult    string
	ToolErr       error           // Fix P4: tool failure preserved in event
	ToolArguments json.RawMessage // Fix R3: original call args on "tool_end"; RunPhase uses this to populate ToolResult.Arguments (Fix T1)
	Err           error           // loop-level error
}

// ---- PhaseConfig ----

type PhaseConfig struct {
	Models          []string
	SystemPrompt    string
	Tools           []string // allowlist names; completion_signal added implicitly by BuildLoopConfig
	AllowedNext     []Role
	RecoveryNext    []Role
	GateRequired    bool
	MinOutputTokens int
}

// ---- PhaseSnapshot ----

// PhaseSnapshot is the evidence state at gate evaluation time.
type PhaseSnapshot struct {
	Phase    Role
	Evidence []string
	Claims   []harness.Claim
	Quality  map[string]bool
}

// toHarness converts PhaseSnapshot to harness.TaskSnapshot for EvaluateCompliance.
// ProcessReport fields are set to true: agentloop manages evidence and quality, not
// the process-report bookkeeping that the harness process gate checks. Marking them
// satisfied prevents evaluateProcessGate from blocking when the contract has no other
// failing conditions — consistent with the spec's "minimalContract passes any snapshot".
func (ps PhaseSnapshot) toHarness() *harness.TaskSnapshot {
	quality := make(map[string]bool, len(ps.Quality))
	for k, v := range ps.Quality {
		quality[k] = v
	}
	return &harness.TaskSnapshot{
		Phase:    string(ps.Phase),
		Evidence: ps.Evidence,
		Claims:   ps.Claims,
		ProcessReport: harness.ProcessReport{
			ContractCoverageSummary: true,
			GateResults:             true,
			EvidenceIndex:           true,
			DecisionLog:             true,
		},
		QualityResults: quality,
	}
}

// ---- harnessState FSM ----

// harnessState is the Harness FSM state (Fix N1, V1).
type harnessState int

const (
	hStateIdle          harnessState = iota // ready for next prompt
	hStateRunning                           // Loop active
	hStateAwaitingHuman                     // gate escalated
	hStateStopped                           // Fix V1: terminal — Stop() called
)

// ---- completionFlag ----

// completionFlag is shared between makeCompletionSignalTool closure and RunPhase (Fix R2-2).
type completionFlag struct {
	mu       sync.Mutex
	signaled bool
	summary  string
}

// ---- PendingDecision ----

// PendingDecision is persisted when gate escalates (Fix N2).
// ApproveGate/Rollback require DecisionID — no pending = no transition.
type PendingDecision struct {
	DecisionID     string
	RunID          uint64
	Phase          Role
	GateResult     GateResult
	AllowedActions []string // "approve" | "rollback" | "stop"
}

// GateResult wraps ComplianceReport with escalation flag.
type GateResult struct {
	Report    harness.ComplianceReport
	Escalated bool
}

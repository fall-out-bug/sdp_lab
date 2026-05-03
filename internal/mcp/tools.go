package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ToolCapability classifies whether a tool is read-only or write-capable.
// Read tools inspect data without side effects. Write tools mutate state
// (filesystem, Beads, Git, deployment targets, etc.).
type ToolCapability string

const (
	// CapabilityRead marks a tool as read-only (no side effects).
	CapabilityRead ToolCapability = "read"
	// CapabilityWrite marks a tool as write-capable (may mutate state).
	CapabilityWrite ToolCapability = "write"
)

// toolPolicyRecord captures the policy classification for a single MCP tool.
type toolPolicyRecord struct {
	name       string
	capability ToolCapability
}

// sdpToolPolicy is the authoritative mapping of SDP-controlled MCP tools
// to their capabilities. The policy is hardcoded here (not derived from
// untrusted tool descriptions or resource text) so that untrusted content
// cannot change a tool's capability classification.
//
// Write-capable tools: beads create/close (mutate tracker state),
// bootstrap (writes files to disk), index build (writes index DB).
// Read-only tools: scout, architect, metrics, spec, index query/find/deps,
// dispatch (routes but does not mutate), beads list (read-only query).
var sdpToolPolicy = []toolPolicyRecord{
	{"sdp_scout", CapabilityRead},
	{"sdp_architect", CapabilityRead},
	{"sdp_metrics", CapabilityRead},
	{"sdp_spec", CapabilityRead},
	{"sdp_bootstrap", CapabilityWrite},
	{"sdp_index_build", CapabilityWrite},
	{"sdp_index_query", CapabilityRead},
	{"sdp_index_find", CapabilityRead},
	{"sdp_index_deps", CapabilityRead},
	{"sdp_dispatch", CapabilityRead},
	{"sdp_beads_create", CapabilityWrite},
	{"sdp_beads_close", CapabilityWrite},
	{"sdp_beads_list", CapabilityRead},
}

// ToolPolicy provides read/write classification and authorization checks
// for SDP-controlled MCP tools. It is the enforcement point for F164
// write-tool policy: untrusted content cannot authorize write calls, and
// read-then-write chains require trusted authorization for the write step.
type ToolPolicy struct {
	records map[string]ToolCapability
}

// NewToolPolicy builds a ToolPolicy from the authoritative SDP tool list.
// It validates that there are no duplicate or ambiguous names.
func NewToolPolicy() (*ToolPolicy, error) {
	tp := &ToolPolicy{records: make(map[string]ToolCapability, len(sdpToolPolicy))}
	for _, r := range sdpToolPolicy {
		if _, exists := tp.records[r.name]; exists {
			return nil, fmt.Errorf("duplicate tool name in policy: %s", r.name)
		}
		tp.records[r.name] = r.capability
	}
	return tp, nil
}

// Classify returns the capability of a tool. Returns ("", false) for
// unknown tools.
func (tp *ToolPolicy) Classify(name string) (ToolCapability, bool) {
	cap, ok := tp.records[name]
	return cap, ok
}

// IsWrite returns true if the tool is classified as write-capable.
// Returns false for unknown tools (fail-closed for write checks).
func (tp *ToolPolicy) IsWrite(name string) bool {
	cap, ok := tp.records[name]
	if !ok {
		return false
	}
	return cap == CapabilityWrite
}

// IsRead returns true if the tool is classified as read-only.
// Returns false for unknown tools.
func (tp *ToolPolicy) IsRead(name string) bool {
	cap, ok := tp.records[name]
	if !ok {
		return false
	}
	return cap == CapabilityRead
}

// AllTools returns all registered tool names and their capabilities.
func (tp *ToolPolicy) AllTools() map[string]ToolCapability {
	result := make(map[string]ToolCapability, len(tp.records))
	for k, v := range tp.records {
		result[k] = v
	}
	return result
}

// AuthorizeWrite checks whether a write-capable tool call is authorized.
// It enforces the F164 policy: write calls require trusted authorization.
// source must be "trusted" for the write to proceed; any other value
// (including untrusted content like repo files, resource text, or
// tool descriptions) results in denied authorization.
func (tp *ToolPolicy) AuthorizeWrite(toolName, source string) error {
	cap, ok := tp.records[toolName]
	if !ok {
		return fmt.Errorf("tool %q not found in policy registry", toolName)
	}
	if cap == CapabilityRead {
		// Read tools don't need write authorization.
		return nil
	}
	// Write-capable tool: source must be trusted.
	if source != "trusted" {
		return fmt.Errorf("write-capable tool %q requires trusted authorization; source=%q is not trusted", toolName, source)
	}
	return nil
}

// ValidateChain checks a sequence of tool calls for read-then-write chains.
// A read-then-write chain is valid only if the write step has trusted
// authorization. This prevents untrusted content read by a read tool from
// inducing an unauthorized write call.
func (tp *ToolPolicy) ValidateChain(calls []ChainCall) error {
	for i, call := range calls {
		if !tp.IsWrite(call.ToolName) {
			continue
		}
		// Check if there was a prior read call in the chain.
		hasPriorRead := false
		for j := 0; j < i; j++ {
			if tp.IsRead(calls[j].ToolName) {
				hasPriorRead = true
				break
			}
		}
		if hasPriorRead && call.Source != "trusted" {
			return fmt.Errorf("read-then-write chain: %s (write) after prior read requires trusted authorization; source=%q", call.ToolName, call.Source)
		}
		// Even without a prior read, standalone writes need trusted source.
		if call.Source != "trusted" {
			return fmt.Errorf("write-capable tool %q requires trusted authorization; source=%q", call.ToolName, call.Source)
		}
	}
	return nil
}

// ChainCall represents a single tool call in a chain for validation.
type ChainCall struct {
	ToolName string
	Source   string // "trusted" or an untrusted source label
}

// ValidateNoDuplicates checks a list of tool records for duplicate names.
// Returns an error listing all duplicates found.
func ValidateNoDuplicates(tools []toolPolicyRecord) error {
	seen := make(map[string]bool)
	var dupes []string
	for _, t := range tools {
		if seen[t.name] {
			dupes = append(dupes, t.name)
		}
		seen[t.name] = true
	}
	if len(dupes) > 0 {
		return fmt.Errorf("duplicate tool names: %s", strings.Join(dupes, ", "))
	}
	return nil
}

// CommandExecutor abstracts running CLI commands. Production code uses
// realExecutor; tests inject mockExecutor.
type CommandExecutor interface {
	// Run executes the sdp CLI with the given subcommand and arguments.
	Run(ctx context.Context, args ...string) ([]byte, error)

	// RunCustom executes a named binary with the given arguments.
	// This is used for non-sdp CLIs like sdp-dispatch or bd.
	RunCustom(ctx context.Context, binary string, args ...string) ([]byte, error)
}

// Ensure ToolPolicy is always initialized when the package loads.
var defaultToolPolicy *ToolPolicy

func init() {
	var err error
	defaultToolPolicy, err = NewToolPolicy()
	if err != nil {
		panic("failed to initialize tool policy: " + err.Error())
	}
}

// DefaultToolPolicy returns the singleton ToolPolicy initialized at package load.
func DefaultToolPolicy() *ToolPolicy {
	return defaultToolPolicy
}

// realExecutor runs actual CLI commands via exec.CommandContext.
type realExecutor struct {
	binaryPath string
	workDir    string // working directory for spawned processes (repo root)
}

func (r *realExecutor) Run(ctx context.Context, args ...string) ([]byte, error) {
	return r.RunCustom(ctx, r.binaryPath, args...)
}

func (r *realExecutor) RunCustom(ctx context.Context, binary string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = r.workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s: %w\n%s", binary, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

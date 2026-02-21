package agent

import (
	"path/filepath"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/llm"
)

// Context holds the full SDP agent execution context.
// Every swarm agent must create and use Context for traceability.
type Context struct {
	// Identity
	AgentID   string
	Role      string
	ProjectID string
	RunID     string
	IssueID   string

	// SDP Protocol Components
	Bus      bus.Bus
	Beads    *beads.Adapter
	Boundary llm.BoundarySpec
	Policy   *PolicyContext

	// Traceability
	Trace      *TraceEmitter
	Evidence   *EvidenceCollector
	Provenance *ProvenanceSigner

	// Extensibility
	Hooks  *HookRegistry
	Skills *SkillRegistry

	// Workspace
	WorkDir string
}

// Config holds options for building AgentContext.
type Config struct {
	AgentID   string
	Role      string
	ProjectID string
	RunID     string
	IssueID   string
	WorkDir   string
	Bus       bus.Bus
	Beads     *beads.Adapter
	Boundary  llm.BoundarySpec
}

// NewContext creates an AgentContext from config.
func NewContext(cfg Config) (*Context, error) {
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
	}
	workDir, err := filepath.Abs(cfg.WorkDir)
	if err != nil {
		return nil, err
	}

	policyCtx := NewPolicyContext(cfg.Role)
	trace := NewTraceEmitter(cfg.Bus, cfg.ProjectID, cfg.RunID, cfg.AgentID, cfg.Role, workDir)
	evidence := NewEvidenceCollector(workDir)
	provenance := NewProvenanceSigner(cfg.AgentID, cfg.Role)
	hooks := NewHookRegistry(cfg.Role)
	skills := NewSkillRegistry(cfg.Role, workDir)

	return &Context{
		AgentID:    cfg.AgentID,
		Role:       cfg.Role,
		ProjectID:  cfg.ProjectID,
		RunID:      cfg.RunID,
		IssueID:    cfg.IssueID,
		Bus:        cfg.Bus,
		Beads:      cfg.Beads,
		Boundary:   cfg.Boundary,
		Policy:     policyCtx,
		Trace:      trace,
		Evidence:   evidence,
		Provenance: provenance,
		Hooks:      hooks,
		Skills:     skills,
		WorkDir:    workDir,
	}, nil
}

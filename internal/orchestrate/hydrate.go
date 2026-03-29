package orchestrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/prompt"
	"sdp_dev/internal/sdputil"
	"sdp_dev/internal/session"

)

const contextPacketPath = ".sdp/context-packet.json"

// ContextPacket is the pre-hydrated context written before each LLM invocation.
// All fields are sourced deterministically (file read, git status, bd show — no LLM).
type ContextPacket struct {
	Workstream         string            `json:"workstream"`
	AcceptanceCriteria []string          `json:"acceptance_criteria"`
	ScopeFiles         []string          `json:"scope_files"`
	Checkpoint         *Checkpoint       `json:"checkpoint,omitempty"`
	Dependencies       map[string]string `json:"dependencies,omitempty"`
	QualityGates       string            `json:"quality_gates"`
	DriftStatus        string            `json:"drift_status"`
	MemoryContext      string            `json:"memory_context,omitempty"`
}

// Hydrate gathers all context deterministically and writes .sdp/context-packet.json.
// Hydration failure blocks LLM invocation (fail-safe). Call before RunBuildPhase or RunReviewPhase.
func Hydrate(projectRoot, featureID, wsID string, cp *Checkpoint) (*ContextPacket, error) {
	if err := sdputil.ValidateWSID(wsID); err != nil {
		return nil, err
	}
	pkt := &ContextPacket{}

	wsPath := filepath.Join(projectRoot, "docs", "workstreams", "backlog", wsID+".md")
	wsContent, err := os.ReadFile(wsPath)
	if err != nil {
		return nil, fmt.Errorf("read workstream %s: %w", wsPath, err)
	}
	pkt.Workstream = string(wsContent)
	pkt.AcceptanceCriteria, pkt.ScopeFiles = parseWorkstreamSections(string(wsContent))
	pkt.Checkpoint = cp

	deps := parseDependsOn(string(wsContent))
	if len(deps) > 0 {
		pkt.Dependencies = make(map[string]string)
		for _, dep := range deps {
			beadsID := wsIDToBeadsID(projectRoot, dep)
			if beadsID != "" {
				out, err := bdShow(projectRoot, beadsID)
				if err != nil {
					pkt.Dependencies[dep] = fmt.Sprintf("ERROR: read dependency %s (%s): %v", dep, beadsID, err)
					continue
				}
				pkt.Dependencies[dep] = out
			}
		}
	}

	agentsPath := filepath.Join(projectRoot, "AGENTS.md")
	agentsContent, err := os.ReadFile(agentsPath)
	if err != nil {
		return nil, fmt.Errorf("read quality gates source %s: %w", agentsPath, err)
	}
	pkt.QualityGates = parseQualityGates(string(agentsContent))
	pkt.DriftStatus, err = gitStatusPorcelain(projectRoot)
	if err != nil {
		pkt.DriftStatus = fmt.Sprintf("ERROR: collect drift status: %v", err)
	}
	pkt.MemoryContext, err = session.LoadMemoryContext(projectRoot)
	if err != nil {
		pkt.MemoryContext = fmt.Sprintf("ERROR: load memory context: %v", err)
	}

	if err := pkt.Validate(); err != nil {
		return nil, fmt.Errorf("context packet validation: %w", err)
	}

	sdpDir := filepath.Join(projectRoot, ".sdp")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir .sdp: %w", err)
	}
	path := filepath.Join(projectRoot, contextPacketPath)
	if err := WriteContextPacket(path, pkt); err != nil {
		return nil, err
	}
	return pkt, nil
}

// HydrateForReview gathers feature-level context when no single wsID applies (review phase).
func HydrateForReview(projectRoot, featureID string, cp *Checkpoint, workstreams []string) (*ContextPacket, error) {
	if len(workstreams) == 0 {
		return nil, fmt.Errorf("no workstreams for feature %s", featureID)
	}
	pkt, err := Hydrate(projectRoot, featureID, workstreams[0], cp)
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(workstreams); i++ {
		if err := sdputil.ValidateWSID(workstreams[i]); err != nil {
			return nil, err
		}
		p := filepath.Join(projectRoot, "docs", "workstreams", "backlog", workstreams[i]+".md")
		if b, err := os.ReadFile(p); err == nil {
			pkt.Workstream += "\n\n---\n\n" + string(b)
		}
	}
	return pkt, nil
}

// Validate checks required fields. Returns error if packet is invalid.
func (p *ContextPacket) Validate() error {
	if p.Workstream == "" {
		return fmt.Errorf("workstream is required")
	}
	if p.QualityGates == "" {
		return fmt.Errorf("quality_gates is required")
	}
	return nil
}

// WriteContextPacket writes the packet to disk (atomic).
func WriteContextPacket(path string, pkt *ContextPacket) error {
	return sdputil.AtomicWriteJSON(path, pkt)
}

// LoadContextPacket reads the packet from disk. Returns nil if file does not exist.
func LoadContextPacket(projectRoot string) (*ContextPacket, error) {
	path := filepath.Join(projectRoot, contextPacketPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var pkt ContextPacket
	if err := sdputil.UnmarshalJSON(data, &pkt); err != nil {
		return nil, fmt.Errorf("parse context packet: %w", err)
	}
	return &pkt, nil
}

// FormatForPrompt returns the packet as a string suitable for injection into the LLM prompt.
func (p *ContextPacket) FormatForPrompt() string {
	var b strings.Builder
	b.WriteString("\n\n## Context Packet (pre-hydrated)\n\n")
	b.WriteString("### Workstream\n\n")
	b.WriteString(p.Workstream)
	b.WriteString("\n\n")
	b.WriteString(prompt.AcceptanceCriteriaSection(p.AcceptanceCriteria))
	b.WriteString(prompt.ScopeFilesSection(p.ScopeFiles))
	b.WriteString("### Quality Gates\n\n")
	b.WriteString(p.QualityGates)
	b.WriteString("\n\n### Drift Status (git status --porcelain)\n\n")
	b.WriteString(p.DriftStatus)
	if p.DriftStatus == "" {
		b.WriteString("(clean)\n")
	}
	if strings.TrimSpace(p.MemoryContext) != "" {
		b.WriteString("\n\n### Memory Context\n\n")
		b.WriteString(p.MemoryContext)
	}
	return b.String()
}

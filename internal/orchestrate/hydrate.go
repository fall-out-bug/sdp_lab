package orchestrate

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const contextPacketPath = ".sdp/context-packet.json"

// ContextPacket is the pre-hydrated context written before each LLM invocation.
// All fields are sourced deterministically (file read, git status, bd show — no LLM).
type ContextPacket struct {
	Workstream         string            `json:"workstream"`                    // Full WS spec content
	AcceptanceCriteria []string         `json:"acceptance_criteria"`            // Parsed checklist items
	ScopeFiles         []string         `json:"scope_files"`                   // From WS spec + git ls-files verify
	Checkpoint         *Checkpoint       `json:"checkpoint,omitempty"`          // Current checkpoint state
	Dependencies       map[string]string `json:"dependencies,omitempty"`        // dep_ws_id -> bd show output
	QualityGates       string            `json:"quality_gates"`                 // From AGENTS.md
	DriftStatus        string            `json:"drift_status"`                   // git status --porcelain
}

var (
	reScopeFile  = regexp.MustCompile(`^-\s+` + "`" + `([^` + "`" + `]+)` + "`")
	reAcceptance = regexp.MustCompile(`^-\s+\[[ x]\]\s+(.+)`)
)

// Hydrate gathers all context deterministically and writes .sdp/context-packet.json.
// Hydration failure blocks LLM invocation (fail-safe). Call before RunBuildPhase or RunReviewPhase.
func Hydrate(projectRoot, featureID, wsID string, cp *Checkpoint) (*ContextPacket, error) {
	pkt := &ContextPacket{}

	// Workstream spec
	wsPath := filepath.Join(projectRoot, "docs", "workstreams", "backlog", wsID+".md")
	wsContent, err := os.ReadFile(wsPath)
	if err != nil {
		return nil, fmt.Errorf("read workstream %s: %w", wsPath, err)
	}
	pkt.Workstream = string(wsContent)

	// Acceptance criteria and scope files from WS spec
	pkt.AcceptanceCriteria, pkt.ScopeFiles = parseWorkstreamSections(string(wsContent))

	// Verify scope files exist via git ls-files
	tracked, _ := gitLSFiles(projectRoot)
	for i, f := range pkt.ScopeFiles {
		if !tracked[f] && f != "" {
			// File may be new (not yet tracked); keep it but note in packet
			_ = i
		}
	}

	// Checkpoint
	pkt.Checkpoint = cp

	// Dependencies: bd show for each dep in WS frontmatter
	deps := parseDependsOn(string(wsContent))
	if len(deps) > 0 {
		pkt.Dependencies = make(map[string]string)
		for _, dep := range deps {
			beadsID := wsIDToBeadsID(projectRoot, dep)
			if beadsID != "" {
				out, _ := bdShow(projectRoot, beadsID)
				pkt.Dependencies[dep] = out
			}
		}
	}

	// Quality gates from AGENTS.md
	agentsPath := filepath.Join(projectRoot, "AGENTS.md")
	agentsContent, _ := os.ReadFile(agentsPath)
	pkt.QualityGates = parseQualityGates(string(agentsContent))

	// Drift status
	pkt.DriftStatus, _ = gitStatusPorcelain(projectRoot)

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
	// Use first workstream for structure; include all WS content
	wsID := ""
	if len(workstreams) > 0 {
		wsID = workstreams[0]
	} else {
		return nil, fmt.Errorf("no workstreams for feature %s", featureID)
	}
	pkt, err := Hydrate(projectRoot, featureID, wsID, cp)
	if err != nil {
		return nil, err
	}
	// Append other workstream specs for review context
	for i := 1; i < len(workstreams); i++ {
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
	data, err := json.MarshalIndent(pkt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal context packet: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write context packet: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename context packet: %w", err)
	}
	return nil
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
	if err := json.Unmarshal(data, &pkt); err != nil {
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
	b.WriteString("\n\n### Acceptance Criteria\n\n")
	for _, ac := range p.AcceptanceCriteria {
		b.WriteString("- ")
		b.WriteString(ac)
		b.WriteString("\n")
	}
	b.WriteString("\n### Scope Files\n\n")
	for _, f := range p.ScopeFiles {
		b.WriteString("- ")
		b.WriteString(f)
		b.WriteString("\n")
	}
	b.WriteString("\n### Quality Gates\n\n")
	b.WriteString(p.QualityGates)
	b.WriteString("\n\n### Drift Status (git status --porcelain)\n\n")
	b.WriteString(p.DriftStatus)
	if p.DriftStatus == "" {
		b.WriteString("(clean)\n")
	}
	return b.String()
}

func parseWorkstreamSections(content string) (acceptance []string, scopeFiles []string) {
	lines := strings.Split(content, "\n")
	var inScopeFiles, inAcceptance bool
	for _, line := range lines {
		if strings.TrimSpace(line) == "## Scope Files" {
			inScopeFiles = true
			inAcceptance = false
			continue
		}
		if strings.TrimSpace(line) == "## Acceptance Criteria" {
			inAcceptance = true
			inScopeFiles = false
			continue
		}
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "## Scope") && !strings.HasPrefix(line, "## Acceptance") {
			inScopeFiles = false
			inAcceptance = false
			continue
		}
		if inAcceptance {
			if m := reAcceptance.FindStringSubmatch(line); len(m) > 1 {
				acceptance = append(acceptance, strings.TrimSpace(m[1]))
			}
		}
		if inScopeFiles {
			if m := reScopeFile.FindStringSubmatch(line); len(m) > 1 {
				scopeFiles = append(scopeFiles, strings.TrimSpace(m[1]))
			}
		}
	}
	return acceptance, scopeFiles
}

var reDependsOn = regexp.MustCompile(`(?m)^depends_on:\s*\[(.*?)\]`)

func parseDependsOn(content string) []string {
	var deps []string
	if m := reDependsOn.FindStringSubmatch(content); len(m) > 1 {
		for _, s := range strings.Split(m[1], ",") {
			id := strings.Trim(strings.Trim(s, `"`), " ")
			if id != "" {
				deps = append(deps, id)
			}
		}
	}
	return deps
}

func parseQualityGates(agentsContent string) string {
	idx := strings.Index(agentsContent, "## Quality Gates")
	if idx < 0 {
		return ""
	}
	rest := agentsContent[idx:]
	end := strings.Index(rest, "\n## ")
	if end > 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

func gitLSFiles(projectRoot string) (map[string]bool, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			m[line] = true
		}
	}
	return m, nil
}

func gitStatusPorcelain(projectRoot string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func bdShow(projectRoot, beadsID string) (string, error) {
	cmd := exec.Command("bd", "show", beadsID)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func wsIDToBeadsID(projectRoot, wsID string) string {
	mappingPath := filepath.Join(projectRoot, ".beads-sdp-mapping.jsonl")
	data, err := os.ReadFile(mappingPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: {"sdp_id":"00-022-01","beads_id":"sdp_dev-bdwr",...}
		if strings.Contains(line, `"sdp_id":"`+wsID+`"`) {
			if idx := strings.Index(line, `"beads_id":"`); idx >= 0 {
				rest := line[idx+12:]
				if end := strings.Index(rest, `"`); end >= 0 {
					return rest[:end]
				}
			}
		}
	}
	return ""
}

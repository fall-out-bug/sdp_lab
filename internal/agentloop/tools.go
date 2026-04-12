package agentloop

// ToolRegistry holds all available tools and filters them per phase allowlist.
// Static for MVP — dynamic registration is backlog (I-tool-registry).
// Fix N6: ToolRegistry never contains "completion_signal".
//   BuildLoopConfig injects it explicitly with the phase-specific completionFlag closure.
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a ToolRegistry from the given tool slice.
// Duplicate names are silently last-wins (later entry overwrites earlier).
func NewToolRegistry(tools []Tool) *ToolRegistry {
	m := make(map[string]Tool, len(tools))
	for _, t := range tools {
		m[t.Name] = t
	}
	return &ToolRegistry{tools: m}
}

// ForPhase returns only the tools whose names appear in cfg.Tools (the phase allowlist).
// Loop receives this already-filtered slice — it cannot call tools outside the allowlist.
// Missing tools (in allowlist but not in registry) are silently omitted.
func (tr *ToolRegistry) ForPhase(cfg PhaseConfig) []Tool {
	result := make([]Tool, 0, len(cfg.Tools))
	for _, name := range cfg.Tools {
		if t, ok := tr.tools[name]; ok {
			result = append(result, t)
		}
	}
	return result
}

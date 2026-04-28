package executor

import (
	"os"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/augmentation"
)

const defaultOmOAgent = "sisyphus"

// OmORole maps SDP phases to OmO agent names.
type OmORole struct {
	Phase       string // SDP phase/task type
	Agent       string // OmO agent name
	Description string
}

var PhaseRoleMap = []OmORole{
	{Phase: "build", Agent: "hephaestus", Description: "Implementation"},
	{Phase: "fix", Agent: "hephaestus", Description: "Bug fix implementation"},
	{Phase: "refactor", Agent: "hephaestus", Description: "Code refactoring"},
	{Phase: "feature", Agent: "hephaestus", Description: "New feature implementation"},
	{Phase: "review", Agent: "momus", Description: "Code review"},
	{Phase: "qa", Agent: "oracle", Description: "Quality assurance testing"},
	{Phase: "plan", Agent: "metis", Description: "Planning and design"},
	{Phase: "explore", Agent: "explore", Description: "Research and exploration"},
	{Phase: "debug", Agent: defaultOmOAgent, Description: "Debugging"},
}

// ResolveAgent returns the OmO agent name for the given phase.
// SDP_DEFAULT_AGENT overrides all routing when set.
func ResolveAgent(phase string) string {
	if override := strings.TrimSpace(os.Getenv("SDP_DEFAULT_AGENT")); override != "" {
		return override
	}

	normalized := strings.ToLower(strings.TrimSpace(phase))
	if normalized == "" {
		return defaultOmOAgent
	}
	if role, ok := augmentation.ResolveDefaultRole(normalized); ok && strings.TrimSpace(role.Agent) != "" {
		return role.Agent
	}
	for _, role := range PhaseRoleMap {
		if normalized == role.Phase {
			return role.Agent
		}
	}
	return defaultOmOAgent
}

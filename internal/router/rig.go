package router

// Rig represents a tool/workflow configuration for SDP execution.
type Rig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Phases      []string `json:"phases"` // which phases this rig supports
}

// defaultRigs defines the built-in rig configurations.
var defaultRigs = map[string]Rig{
	"sdp-full": {Name: "sdp-full", Description: "Full SDP pipeline", Phases: []string{"discovery", "design", "build", "review", "qa"}},
	"sdp-lite": {Name: "sdp-lite", Description: "Lightweight SDP (skip discovery)", Phases: []string{"design", "build", "review"}},
	"manual":   {Name: "manual", Description: "Manual execution", Phases: []string{"build"}},
}

// selectRig picks a rig based on task type and project config.
// Hotfixes always use manual (fastest path). Bugfixes use sdp-lite.
// Everything else respects the project default, falling back to sdp-full.
func selectRig(taskType string, projectDefault string) string {
	switch taskType {
	case "hotfix":
		return "manual"
	case "bugfix":
		return "sdp-lite"
	default:
		if projectDefault != "" {
			return projectDefault
		}
		return "sdp-full"
	}
}

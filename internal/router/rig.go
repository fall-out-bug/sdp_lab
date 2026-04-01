package router

// Rig represents a tool/workflow configuration for SDP execution.
type Rig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Phases      []string `json:"phases"` // which phases this rig supports
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

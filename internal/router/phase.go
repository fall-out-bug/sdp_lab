package router

// InferEntryPhase determines the starting phase based on task type and signals.
// The logic follows SDP conventions:
//   - hotfix/bugfix: skip directly to build
//   - feature: depends on available artifacts (requirements, design)
//   - refactor: start at design (architecture decisions needed)
//   - unknown: start at discovery (safest default)
func InferEntryPhase(taskType string, hasRequirements, hasDesign bool) string {
	switch taskType {
	case "hotfix", "bugfix":
		return "build"
	case "feature":
		if hasDesign {
			return "build"
		}
		if hasRequirements {
			return "design"
		}
		return "discovery"
	case "refactor":
		return "design"
	default:
		return "discovery"
	}
}

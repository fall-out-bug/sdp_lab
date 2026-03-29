package dispatch

import "os/exec"

// Infrastructure abstracts convoy and sling operations over a backend.
type Infrastructure interface {
	// Name returns "gastown" or "standalone".
	Name() string
	// CreateConvoy creates a named convoy containing the given issues and
	// returns its ID.
	CreateConvoy(name string, issues []string) (convoyID string, err error)
	// ConvoyStatus returns the status of a convoy: "active", "completed", or
	// "unknown".
	ConvoyStatus(id string) (string, error)
	// Sling dispatches an issue to a rig using the named agent.
	Sling(issue, rig, agent string) error
}

// DetectInfrastructure returns a GastownInfra when the "gt" binary is found on
// PATH, and a StandaloneInfra otherwise.
func DetectInfrastructure() Infrastructure {
	if _, err := exec.LookPath("gt"); err == nil {
		return &GastownInfra{}
	}
	return &StandaloneInfra{}
}

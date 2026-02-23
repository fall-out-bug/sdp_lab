package adapter

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LabelProject is the canonical label for project ID. "project" is legacy.
const LabelProject = "sdp.project"

// WorkspaceHealth holds the result of workspace availability checks.
type WorkspaceHealth struct {
	BeadsAvailable    bool   // bd in PATH and .beads/ exists
	BeadsFSMAvailable bool   // beads-fsm in PATH
	Reason            string // why beads unavailable
}

// CheckWorkspaceHealth checks if beads and beads-fsm are usable in workDir.
func CheckWorkspaceHealth(workDir string) WorkspaceHealth {
	h := WorkspaceHealth{}
	if workDir == "" {
		workDir = "."
	}
	if _, err := exec.LookPath("bd"); err != nil {
		h.Reason = "bd not in PATH"
		return h
	}
	beadsDir := filepath.Join(workDir, ".beads")
	if st, err := os.Stat(beadsDir); err != nil || !st.IsDir() {
		h.Reason = ".beads/ absent or not a directory"
		return h
	}
	h.BeadsAvailable = true
	if _, err := exec.LookPath("beads-fsm"); err == nil {
		h.BeadsFSMAvailable = true
	}
	return h
}

// ProjectIDFromLabels returns projectID from labels. Prefer sdp.project, fallback to project.
func ProjectIDFromLabels(labels map[string]string) string {
	if labels == nil {
		return ""
	}
	if v := labels[LabelProject]; strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(labels["project"])
}

// WorkspaceResolver returns the workspace path for a project. Empty projectID -> baseDir/default.
type WorkspaceResolver func(projectID string) string

// NewWorkspaceResolver returns a resolver that maps projectID to baseDir/projectID.
func NewWorkspaceResolver(baseDir string) WorkspaceResolver {
	return func(projectID string) string {
		if strings.TrimSpace(projectID) == "" {
			projectID = "default"
		}
		return filepath.Join(baseDir, projectID)
	}
}

// BeadsFSMAvailable returns true if beads-fsm is in PATH.
func BeadsFSMAvailable() bool {
	_, err := exec.LookPath("beads-fsm")
	return err == nil
}

// HandoffPath returns the relative path to a role's handoff artifact.
// Format: .sdp/handoff/<issueID>/<role>.json
func HandoffPath(issueID, role string) string {
	if issueID == "" || role == "" {
		return ""
	}
	return ".sdp/handoff/" + issueID + "/" + role + ".json"
}

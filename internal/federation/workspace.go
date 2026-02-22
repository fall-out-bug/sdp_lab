package federation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// WorkspaceManager clones/pulls per-project workspaces in a base directory.
type WorkspaceManager struct {
	baseDir string
}

// NewWorkspaceManager creates a WorkspaceManager.
func NewWorkspaceManager(baseDir string) *WorkspaceManager {
	return &WorkspaceManager{baseDir: baseDir}
}

// EnsureWorkspace ensures the project workspace exists and is up to date.
// Returns the absolute path to the workspace. If repoURL is "." or empty, uses baseDir/projectID as-is.
func (w *WorkspaceManager) EnsureWorkspace(projectID, repoURL, repoBranch string) (string, error) {
	workspacePath := filepath.Join(w.baseDir, projectID)
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		return "", fmt.Errorf("mkdir workspace: %w", err)
	}

	if repoURL == "" || repoURL == "." {
		return filepath.Abs(workspacePath)
	}

	gitDir := filepath.Join(workspacePath, ".git")
	if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
		// Already cloned; pull
		cmd := exec.Command("git", "pull", "--rebase")
		cmd.Dir = workspacePath
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git pull: %w: %s", err, string(out))
		}
		return filepath.Abs(workspacePath)
	}

	// Clone
	if repoBranch == "" {
		repoBranch = "main"
	}
	cmd := exec.Command("git", "clone", "--branch", repoBranch, "--single-branch", repoURL, workspacePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone: %w: %s", err, string(out))
	}
	return filepath.Abs(workspacePath)
}

// WorkspacePath returns the path for a project without cloning.
func (w *WorkspaceManager) WorkspacePath(projectID string) string {
	return filepath.Join(w.baseDir, projectID)
}

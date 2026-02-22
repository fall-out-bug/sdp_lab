package federation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sdp_dev/internal/registry"
)

// WorkspaceManager clones/pulls per-project workspaces in a base directory.
// Supports fork repos: origin = fork URL, upstream = source URL.
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
	return w.ensureWorkspace(projectID, repoURL, repoBranch, false, "", "")
}

// EnsureWorkspaceFromProject ensures workspace using full project config (fork support).
func (w *WorkspaceManager) EnsureWorkspaceFromProject(proj *registry.Project) (string, error) {
	upstreamRemote := proj.UpstreamRemote
	if proj.Fork && upstreamRemote == "" {
		upstreamRemote = "upstream"
	}
	return w.ensureWorkspace(proj.ID, proj.RepoURL, proj.RepoBranch, proj.Fork, upstreamRemote, proj.UpstreamURL)
}

func (w *WorkspaceManager) ensureWorkspace(projectID, repoURL, repoBranch string, fork bool, upstreamRemote, upstreamURL string) (string, error) {
	workspacePath := filepath.Join(w.baseDir, projectID)
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		return "", fmt.Errorf("mkdir workspace: %w", err)
	}

	if repoURL == "" || repoURL == "." {
		return filepath.Abs(workspacePath)
	}

	gitDir := filepath.Join(workspacePath, ".git")
	if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
		// Already cloned; pull origin
		if out, err := runGit(workspacePath, "pull", "--rebase"); err != nil {
			return "", fmt.Errorf("git pull: %w: %s", err, string(out))
		}
		if fork && strings.TrimSpace(upstreamURL) != "" && strings.TrimSpace(upstreamRemote) != "" {
			_, _ = runGit(workspacePath, "fetch", upstreamRemote)
		}
		return filepath.Abs(workspacePath)
	}

	// Clone (origin = repoURL)
	if repoBranch == "" {
		repoBranch = "main"
	}
	if out, err := runGit("", "clone", "--branch", repoBranch, "--single-branch", repoURL, workspacePath); err != nil {
		return "", fmt.Errorf("git clone: %w: %s", err, string(out))
	}

	// Fork: add upstream remote and fetch
	if fork && strings.TrimSpace(upstreamURL) != "" && strings.TrimSpace(upstreamRemote) != "" {
		if out, err := runGit(workspacePath, "remote", "add", upstreamRemote, upstreamURL); err != nil {
			if !strings.Contains(string(out), "already exists") {
				return "", fmt.Errorf("git remote add %s: %w: %s", upstreamRemote, err, string(out))
			}
		}
		if out, err := runGit(workspacePath, "fetch", upstreamRemote); err != nil {
			return "", fmt.Errorf("git fetch %s: %w: %s", upstreamRemote, err, string(out))
		}
	}

	return filepath.Abs(workspacePath)
}

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}

// WorkspacePath returns the path for a project without cloning.
func (w *WorkspaceManager) WorkspacePath(projectID string) string {
	return filepath.Join(w.baseDir, projectID)
}

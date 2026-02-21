package orchestrator

import (
	"fmt"
	"os/exec"
	"strings"
)

// Runtime identifies the execution runtime (OpenCode or OpenClaw).
type Runtime string

const (
	RuntimeOpenCode Runtime = "opencode"
	RuntimeOpenClaw Runtime = "openclaw"
)

// DispatchConfig holds parameters for dispatching a task.
type DispatchConfig struct {
	Runtime   Runtime
	Host      string
	Port      string
	IssueID   string
	Namespace string
	InCluster bool
}

// SelectRuntime chooses runtime based on issue labels. For now always OpenCode.
func SelectRuntime(labels []string) Runtime {
	for _, l := range labels {
		if l == "runtime:openclaw" {
			return RuntimeOpenClaw
		}
	}
	return RuntimeOpenCode
}

// Dispatcher executes the dispatch (SSH or in-cluster).
type Dispatcher interface {
	Dispatch(cfg DispatchConfig) error
}

// SSHDispatcher dispatches via SSH to remote host, running orchestrate logic.
type SSHDispatcher struct {
	workDir string
}

// NewSSHDispatcher returns a dispatcher for SSH mode.
func NewSSHDispatcher(workDir string) *SSHDispatcher {
	return &SSHDispatcher{workDir: workDir}
}

// Dispatch runs the orchestrate script on the remote host via SSH.
func (d *SSHDispatcher) Dispatch(cfg DispatchConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("host required for SSH dispatch")
	}
	port := cfg.Port
	if port == "" {
		port = "22"
	}
	args := []string{
		"-p", port, cfg.Host,
		"kubectl", "-n", cfg.Namespace,
		"exec", "deploy/opencode-agent", "--",
		"sh", "-lc",
		"cd /workspace && git rev-parse --is-inside-work-tree >/dev/null && branch=\"${SDP_REPO_BRANCH:-master}\" && git fetch origin \"$branch\" && git rebase FETCH_HEAD && bd sync --import-only >/dev/null",
	}
	cmd := exec.Command("ssh", args...)
	cmd.Dir = d.workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("preflight: %w: %s", err, string(out))
	}

	// Trigger one agent cycle
	cycleCmd := fmt.Sprintf("set -e; lock=/tmp/sdp-orchestrate-%s.lock; if mkdir \"$lock\" 2>/dev/null; then trap 'rmdir \"$lock\"' EXIT; opencode-agent; else echo lock-exists; fi", cfg.IssueID)
	args2 := []string{
		"-p", port, cfg.Host,
		"kubectl", "-n", cfg.Namespace,
		"exec", "deploy/opencode-agent", "--",
		"sh", "-lc", cycleCmd,
	}
	cmd2 := exec.Command("ssh", args2...)
	cmd2.Dir = d.workDir
	out2, err := cmd2.CombinedOutput()
	if err != nil && !strings.Contains(string(out2), "lock-exists") {
		return fmt.Errorf("dispatch: %w: %s", err, string(out2))
	}
	return nil
}

// InClusterDispatcher dispatches by exec into opencode-agent (when orchestrator runs in-cluster).
type InClusterDispatcher struct {
	workDir string
}

// NewInClusterDispatcher returns a dispatcher for in-cluster mode.
func NewInClusterDispatcher(workDir string) *InClusterDispatcher {
	return &InClusterDispatcher{workDir: workDir}
}

// Dispatch runs preflight and opencode-agent via kubectl exec (orchestrator pod has kubeconfig).
func (d *InClusterDispatcher) Dispatch(cfg DispatchConfig) error {
	ns := cfg.Namespace
	if ns == "" {
		ns = "sdp-workers"
	}
	preflight := "cd /workspace && git rev-parse --is-inside-work-tree >/dev/null && branch=\"${SDP_REPO_BRANCH:-master}\" && git fetch origin \"$branch\" && git rebase FETCH_HEAD && bd sync --import-only >/dev/null"
	cmd := exec.Command("kubectl", "-n", ns, "exec", "deploy/opencode-agent", "--", "sh", "-lc", preflight)
	cmd.Dir = d.workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("preflight: %w: %s", err, string(out))
	}
	cycleCmd := fmt.Sprintf("set -e; lock=/tmp/sdp-orchestrate-%s.lock; if mkdir \"$lock\" 2>/dev/null; then trap 'rmdir \"$lock\"' EXIT; opencode-agent; else echo lock-exists; fi", cfg.IssueID)
	cmd2 := exec.Command("kubectl", "-n", ns, "exec", "deploy/opencode-agent", "--", "sh", "-lc", cycleCmd)
	cmd2.Dir = d.workDir
	out2, err := cmd2.CombinedOutput()
	if err != nil && !strings.Contains(string(out2), "lock-exists") {
		return fmt.Errorf("dispatch: %w: %s", err, string(out2))
	}
	return nil
}

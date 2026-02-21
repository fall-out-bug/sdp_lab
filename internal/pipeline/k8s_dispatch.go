package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"sdp_dev/internal/federation"
	"sdp_dev/internal/safeid"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	defaultK8sNamespace   = "sdp-workers"
	opencodeAgentLabel    = "app=opencode-agent"
	pollInterval          = 10 * time.Second
	pollTimeout          = 15 * time.Minute
	terminalStatusClosed  = "closed"
	terminalStatusBlocked = "blocked"
)

// dispatchK8s delegates task execution to the opencode-agent pod in K8s via client-go exec.
func dispatchK8s(ctx context.Context, task federation.FederatedTask) error {
	if err := safeid.ValidateIssueID(task.Issue.ID); err != nil {
		return fmt.Errorf("invalid issue ID: %w", err)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("in-cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("kubernetes client: %w", err)
	}

	ns := os.Getenv("SDP_K8S_NAMESPACE")
	if ns == "" {
		ns = defaultK8sNamespace
	}

	podName, err := findOpencodeAgentPod(ctx, clientset, ns)
	if err != nil {
		return fmt.Errorf("find opencode-agent pod: %w", err)
	}
	log.Printf("k8s_dispatch: issue=%s pod=%s ns=%s", task.Issue.ID, podName, ns)

	// 1. Preflight sync
	preflight := "cd /workspace && git rev-parse --is-inside-work-tree >/dev/null && branch=\"${SDP_REPO_BRANCH:-master}\" && git fetch origin \"$branch\" && git rebase FETCH_HEAD && bd sync --import-only >/dev/null"
	if err := execInPod(ctx, config, clientset, ns, podName, "", preflight); err != nil {
		return fmt.Errorf("preflight: %w", err)
	}
	log.Printf("k8s_dispatch: issue=%s preflight ok", task.Issue.ID)

	// 2. Claim issue (bd update --status in_progress)
	claimCmd := fmt.Sprintf("cd /workspace && bd update %s --status in_progress", task.Issue.ID)
	if err := execInPod(ctx, config, clientset, ns, podName, "", claimCmd); err != nil {
		return fmt.Errorf("claim: %w", err)
	}
	log.Printf("k8s_dispatch: issue=%s claim ok", task.Issue.ID)

	// 3. Trigger single agent cycle with SDP_ISSUE for targeted execution
	cycleCmd := fmt.Sprintf("cd /workspace && SDP_ISSUE=%s opencode-agent", task.Issue.ID)
	if err := execInPod(ctx, config, clientset, ns, podName, "", cycleCmd); err != nil {
		return fmt.Errorf("agent cycle: %w", err)
	}
	log.Printf("k8s_dispatch: issue=%s cycle done, polling status", task.Issue.ID)

	// 4. Poll status until terminal (closed or blocked)
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		status, err := getIssueStatus(ctx, config, clientset, ns, podName, task.Issue.ID)
		if err != nil {
			return fmt.Errorf("poll status: %w", err)
		}
		if status == terminalStatusClosed || status == terminalStatusBlocked {
			log.Printf("k8s_dispatch: issue=%s terminal status=%s", task.Issue.ID, status)
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("poll timeout: issue %s did not reach terminal status in %v", task.Issue.ID, pollTimeout)
}

func findOpencodeAgentPod(ctx context.Context, clientset kubernetes.Interface, ns string) (string, error) {
	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: opencodeAgentLabel,
	})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no opencode-agent pod found in %s", ns)
	}
	// Prefer Running pods
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning {
			return p.Name, nil
		}
	}
	return pods.Items[0].Name, nil
}

func execInPod(ctx context.Context, config *rest.Config, clientset kubernetes.Interface, ns, podName, container string, cmd string) error {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(ns).
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"sh", "-c", cmd},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return fmt.Errorf("%w: stderr=%s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func getIssueStatus(ctx context.Context, config *rest.Config, clientset kubernetes.Interface, ns, podName, issueID string) (string, error) {
	cmd := fmt.Sprintf("cd /workspace && bd show %s --json 2>/dev/null || echo '{}'", issueID)
	var stdout bytes.Buffer
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(ns).
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: []string{"sh", "-c", cmd},
			Stdout:  true,
			Stderr:  true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", err
	}
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout})
	if err != nil {
		return "", err
	}

	var out struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return "", nil // treat parse error as non-terminal
	}
	return out.Status, nil
}

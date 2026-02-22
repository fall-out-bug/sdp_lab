package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
)

// DeployTrigger holds deployment trigger data from NATS.
type DeployTrigger struct {
	Ref     string `json:"ref"`      // git SHA or tag
	Project string `json:"project"`  // repo/project ID
	Env     string `json:"env"`      // dev, staging, prod
	Subject string `json:"subject"`  // NATS subject
}

// Agent runs the CI/CD deployment pipeline on NATS triggers.
type Agent struct {
	bus              bus.Bus
	registry         string   // e.g. ghcr.io/owner
	imageTag         string   // default tag
	images           []string // image names to build
	workDir          string
	sshHost          string // optional: deploy via SSH
	kubeconfig       string
	beadsAdapter     *beads.Adapter
	mu               sync.Mutex
	deployInProgress bool
}

// AgentConfig holds configuration for the CI/CD agent.
type AgentConfig struct {
	Registry   string   // ghcr.io/owner
	ImageTag   string   // git-SHA or latest
	Images     []string // adapter-controller, feature-orchestrator, etc.
	WorkDir    string
	SSHHost    string
	Kubeconfig string
}

// NewAgent creates a CI/CD agent.
func NewAgent(b bus.Bus, cfg AgentConfig) *Agent {
	if cfg.Registry == "" {
		cfg.Registry = "ghcr.io/fall-out-bug"
	}
	if cfg.ImageTag == "" {
		cfg.ImageTag = "latest"
	}
	if len(cfg.Images) == 0 {
		cfg.Images = []string{
			"adapter-controller", "feature-orchestrator", "swarm-orchestrator",
			"intake-gateway", "registry-agent", "telemetry-analyzer",
		}
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
	}
	return &Agent{
		bus:          b,
		registry:     cfg.Registry,
		imageTag:     cfg.ImageTag,
		images:       cfg.Images,
		workDir:      cfg.WorkDir,
		sshHost:      cfg.SSHHost,
		kubeconfig:   cfg.Kubeconfig,
		beadsAdapter: beads.NewAdapter(cfg.WorkDir),
	}
}

// Run subscribes to deploy triggers and runs the pipeline.
func (a *Agent) Run(ctx context.Context) error {
	_, err := a.bus.Subscribe("sdp.deploy.trigger.>", "cicd-agent", func(env bus.Envelope) {
		trigger := a.parseTrigger(env)
		if trigger.Ref == "" {
			trigger.Ref = a.imageTag
		}
		go a.deploy(ctx, trigger)
	})
	if err != nil {
		return err
	}
	_, err = a.bus.Subscribe("sdp.github.pr.merged", "cicd-agent", func(env bus.Envelope) {
		trigger := a.parsePRMerged(env)
		if trigger != nil {
			go a.deploy(ctx, *trigger)
		}
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) parseTrigger(env bus.Envelope) DeployTrigger {
	var t DeployTrigger
	if len(env.Payload) > 0 {
		_ = json.Unmarshal(env.Payload, &t)
	}
	if t.Ref == "" && env.IssueID != "" {
		t.Ref = env.IssueID
	}
	if t.Project == "" {
		t.Project = env.ProjectID
	}
	if t.Env == "" {
		t.Env = "dev"
	}
	return t
}

func (a *Agent) parsePRMerged(env bus.Envelope) *DeployTrigger {
	var pr struct {
		TargetBranch string `json:"target_branch"`
		MergeCommit  string `json:"merge_commit_sha"`
		Repo         string `json:"repository"`
	}
	if len(env.Payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Payload, &pr); err != nil {
		return nil
	}
	if pr.TargetBranch != "dev" && pr.TargetBranch != "main" && pr.TargetBranch != "master" {
		return nil
	}
	envName := "dev"
	if pr.TargetBranch == "main" || pr.TargetBranch == "master" {
		envName = "prod"
	}
	return &DeployTrigger{
		Ref:     pr.MergeCommit,
		Project: pr.Repo,
		Env:     envName,
	}
}

func (a *Agent) deploy(ctx context.Context, t DeployTrigger) {
	a.mu.Lock()
	if a.deployInProgress {
		a.mu.Unlock()
		log.Printf("cicd-agent: deploy already in progress, skipping")
		return
	}
	a.deployInProgress = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.deployInProgress = false
		a.mu.Unlock()
	}()

	start := time.Now()
	tag := t.Ref
	if len(tag) > 12 {
		tag = "git-" + tag[:12]
	}
	if tag == "" {
		tag = a.imageTag
	}

	a.publishStatus(t, "started", "")
	beadsID := a.createDeployBead(t, tag)
	imageSHAs := make(map[string]string)

	if err := a.buildAndPush(ctx, tag, imageSHAs); err != nil {
		log.Printf("cicd-agent: build/push failed: %v", err)
		a.writeDeployEvidence(t, tag, "failed", err.Error(), imageSHAs, start)
		a.publishStatus(t, "failed", err.Error())
		return
	}
	if err := a.applyAndRollout(ctx, t, tag); err != nil {
		log.Printf("cicd-agent: deploy failed: %v", err)
		a.writeDeployEvidence(t, tag, "failed", err.Error(), imageSHAs, start)
		a.publishStatus(t, "failed", err.Error())
		if rollbackErr := a.rollback(ctx, t); rollbackErr != nil {
			log.Printf("cicd-agent: rollback failed: %v", rollbackErr)
		} else {
			a.publishStatus(t, "rolled_back", "")
		}
		return
	}
	healthOK := a.healthCheck(ctx, t)
	if !healthOK {
		log.Printf("cicd-agent: health check failed")
		_ = a.rollback(ctx, t)
		a.publishStatus(t, "rolled_back", "")
		a.writeDeployEvidence(t, tag, "failed", "health check failed", imageSHAs, start)
		a.publishStatus(t, "failed", "health check failed")
		return
	}
	a.writeDeployEvidence(t, tag, "succeeded", "", imageSHAs, start)
	a.publishStatus(t, "succeeded", "")
	if beadsID != "" {
		_ = a.beadsAdapter.Close(beadsID, "Deploy succeeded")
	}
}

func (a *Agent) buildAndPush(ctx context.Context, tag string, imageSHAs map[string]string) error {
	prefix := a.registry + "/sdp-dev-"
	for _, name := range a.images {
		dockerfile := filepath.Join("deploy", "images", name, "Dockerfile")
		if _, err := os.Stat(filepath.Join(a.workDir, dockerfile)); os.IsNotExist(err) {
			continue
		}
		image := prefix + name + ":" + tag
		cmd := exec.CommandContext(ctx, "docker", "build", "-t", image, "-f", dockerfile, ".")
		cmd.Dir = a.workDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("docker build %s: %w: %s", name, err, string(out))
		}
		cmd = exec.CommandContext(ctx, "docker", "push", image)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("docker push %s: %w: %s", image, err, string(out))
		}
		if imageSHAs != nil {
			if sha := a.dockerInspectSHA(ctx, image); sha != "" {
				imageSHAs[image] = sha
			}
		}
	}
	return nil
}

func (a *Agent) dockerInspectSHA(ctx context.Context, image string) string {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Id}}", image)
	cmd.Dir = a.workDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (a *Agent) applyAndRollout(ctx context.Context, t DeployTrigger, tag string) error {
	manifestsDir := filepath.Join(a.workDir, "deploy", "k8s", "control")
	// Wire image tag from deploy trigger: kustomize edit set image before apply (q65)
	prefix := a.registry + "/sdp-dev-"
	for _, name := range a.images {
		kustomizeName := "sdp/" + name
		fullImage := prefix + name + ":" + tag
		cmd := exec.CommandContext(ctx, "kustomize", "edit", "set", "image", kustomizeName+"="+fullImage)
		cmd.Dir = manifestsDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("kustomize edit set image %s: %w: %s", kustomizeName, err, string(out))
		}
	}
	if a.sshHost != "" {
		return a.applyAndRolloutSSH(ctx, manifestsDir)
	}
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-k", manifestsDir)
	cmd.Dir = a.workDir
	cmd.Env = a.kubectlEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kubectl apply: %w: %s", err, string(out))
	}
	cmd = exec.CommandContext(ctx, "kubectl", "rollout", "status", "deployment", "-n", "sdp-control", "--timeout=300s")
	cmd.Dir = a.workDir
	cmd.Env = a.kubectlEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kubectl rollout: %w: %s", err, string(out))
	}
	return nil
}

// applyAndRolloutSSH runs kustomize build locally and pipes to kubectl apply on remote host.
func (a *Agent) applyAndRolloutSSH(ctx context.Context, manifestsDir string) error {
	buildCmd := exec.CommandContext(ctx, "kustomize", "build", manifestsDir)
	buildCmd.Dir = a.workDir
	manifests, err := buildCmd.Output()
	if err != nil {
		return fmt.Errorf("kustomize build: %w", err)
	}
	kubectlCmd := "kubectl apply -f -"
	if a.kubeconfig != "" {
		kubectlCmd = "KUBECONFIG=" + a.kubeconfig + " kubectl apply -f -"
	}
	sshCmd := exec.CommandContext(ctx, "ssh", a.sshHost, kubectlCmd)
	sshCmd.Stdin = strings.NewReader(string(manifests))
	if out, err := sshCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh kubectl apply: %w: %s", err, string(out))
	}
	rolloutCmd := "kubectl rollout status deployment -n sdp-control --timeout=300s"
	if a.kubeconfig != "" {
		rolloutCmd = "KUBECONFIG=" + a.kubeconfig + " " + rolloutCmd
	}
	sshCmd = exec.CommandContext(ctx, "ssh", a.sshHost, rolloutCmd)
	if out, err := sshCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh kubectl rollout: %w: %s", err, string(out))
	}
	return nil
}

func (a *Agent) kubectlEnv() []string {
	env := os.Environ()
	if a.kubeconfig != "" {
		env = append(env, "KUBECONFIG="+a.kubeconfig)
	}
	return env
}

func (a *Agent) healthCheck(ctx context.Context, t DeployTrigger) bool {
	kubectlCmd := "kubectl get pods -n sdp-control -l app=feature-orchestrator -o jsonpath={.items[0].status.phase}"
	if a.kubeconfig != "" {
		kubectlCmd = "KUBECONFIG=" + a.kubeconfig + " " + kubectlCmd
	}
	var cmd *exec.Cmd
	if a.sshHost != "" {
		cmd = exec.CommandContext(ctx, "ssh", a.sshHost, kubectlCmd)
	} else {
		cmd = exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", "sdp-control", "-l", "app=feature-orchestrator", "-o", "jsonpath={.items[0].status.phase}")
		cmd.Env = a.kubectlEnv()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "Running"
}

func (a *Agent) rollback(ctx context.Context, t DeployTrigger) error {
	rollbackCmd := "kubectl rollout undo deployment -n sdp-control"
	if a.kubeconfig != "" {
		rollbackCmd = "KUBECONFIG=" + a.kubeconfig + " " + rollbackCmd
	}
	var cmd *exec.Cmd
	if a.sshHost != "" {
		cmd = exec.CommandContext(ctx, "ssh", a.sshHost, rollbackCmd)
	} else {
		cmd = exec.CommandContext(ctx, "kubectl", "rollout", "undo", "deployment", "-n", "sdp-control")
		cmd.Env = a.kubectlEnv()
	}
	_, err := cmd.CombinedOutput()
	return err
}

func (a *Agent) publishStatus(t DeployTrigger, status, reason string) {
	if a.bus == nil {
		return
	}
	subject := "sdp.deploy." + t.Env + "." + status
	payload, _ := json.Marshal(map[string]string{
		"ref": t.Ref, "project": t.Project, "env": t.Env, "reason": reason,
	})
	env := bus.Envelope{
		IssueID:       t.Ref,
		ProjectID:     t.Project,
		ArtifactClass: "deploy",
		Phase:         status,
		Role:          "cicd-agent",
		Payload:       payload,
	}
	_ = a.bus.Publish(subject, env)
}

func (a *Agent) createDeployBead(t DeployTrigger, tag string) string {
	if a.beadsAdapter == nil {
		return ""
	}
	title := fmt.Sprintf("Deploy %s to %s (%s)", t.Project, t.Env, tag)
	id, err := a.beadsAdapter.Create(beads.CreateOpts{
		Title: title,
		Type:  "chore",
		Labels: []string{"source:cicd-agent", "deploy"},
	})
	if err != nil {
		log.Printf("cicd-agent: beads create: %v", err)
		return ""
	}
	return id
}

func (a *Agent) writeDeployEvidence(t DeployTrigger, tag, status, reason string, imageSHAs map[string]string, start time.Time) {
	evDir := filepath.Join(a.workDir, ".sdp", "evidence")
	_ = os.MkdirAll(evDir, 0755)
	evID := "deploy-" + strings.ReplaceAll(tag, ":", "-") + "-" + t.Env
	evPath := filepath.Join(evDir, evID+".json")
	duration := time.Since(start).Seconds()
	ev := map[string]any{
		"intent": map[string]any{
			"ref": t.Ref, "project": t.Project, "env": t.Env, "tag": tag,
		},
		"execution": map[string]any{
			"status":   status,
			"reason":   reason,
			"duration": duration,
			"image_shas": imageSHAs,
		},
		"verification": map[string]any{
			"health_ok": status == "succeeded",
		},
	}
	b, _ := json.MarshalIndent(ev, "", "  ")
	_ = os.WriteFile(evPath, b, 0644)
}


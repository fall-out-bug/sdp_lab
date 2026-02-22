// feature-orchestrator polls ready tasks, prioritizes, creates AgentRun CRDs.
// Replaces swarm-orchestrator dispatch with K8s-native AgentRun creation.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/adapter"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/observability"
	"sdp_dev/internal/registry"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	if _, err := observability.SetupTracing("feature-orchestrator"); err != nil {
		log.Printf("tracing setup: %v (continuing)", err)
	}

	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	workspaceBase := flag.String("workspace", "/workspaces", "base dir for project workspaces")
	pollInterval := flag.Duration("poll", envDuration("SDP_POLL_INTERVAL", 30*time.Second), "Poll interval")
	maxConcurrent := flag.Int("max", envInt("SDP_MAX_CONCURRENT", 3), "Max concurrent AgentRuns to create per cycle")
	projectFilter := flag.String("projects", os.Getenv("SDP_PROJECT_FILTER"), "Comma-separated project IDs to include (empty = all)")
	namespace := flag.String("namespace", envStr("SDP_AGENTRUN_NAMESPACE", "sdp-workers"), "Kubernetes namespace for AgentRuns")
	flag.Parse()

	if *natsURL == "" {
		log.Fatal("NATS_URL or -nats required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	b, err := bus.ConnectAndProvision(ctx, *natsURL)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer b.Close()

	registryPath := os.Getenv("SDP_REGISTRY_PATH")
	if registryPath == "" {
		registryPath = "specs/project-registry.yaml"
	}
	store := registry.NewStore(registry.StoreConfig{RegistryPath: registryPath})
	if err := store.Load(); err != nil {
		log.Printf("registry load: %v (continuing)", err)
	}

	ws := federation.NewWorkspaceManager(*workspaceBase)
	agg := federation.NewAggregator(b, store, ws)
	go func() {
		if err := agg.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("aggregator: %v", err)
		}
	}()

	cfg, err := getKubeConfig()
	if err != nil {
		log.Fatalf("kube config: %v", err)
	}
	k8s, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("k8s client: %v", err)
	}

	lockMgr := adapter.NewLeaseLockManager(k8s, *namespace)
	filter := parseProjectFilter(*projectFilter)

	log.Printf("feature-orchestrator running (poll=%v, max=%d)", *pollInterval, *maxConcurrent)
	ticker := time.NewTicker(*pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dispatch(ctx, DispatchConfig{
				K8s:       k8s,
				Agg:       agg,
				LockMgr:   lockMgr,
				Store:     store,
				Filter:    filter,
				Namespace: *namespace,
				Max:       *maxConcurrent,
			})
		}
	}
}

func parseProjectFilter(s string) map[string]bool {
	out := make(map[string]bool)
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out[p] = true
		}
	}
	return out
}

// DispatchConfig holds parameters for the dispatch loop.
type DispatchConfig struct {
	K8s       client.Client
	Agg       *federation.Aggregator
	LockMgr   adapter.RunLock
	Store     *registry.Store
	Filter    map[string]bool
	Namespace string
	Max       int
}

func dispatch(ctx context.Context, cfg DispatchConfig) {
	tasks := cfg.Agg.ReadyAcrossProjects(cfg.Max * 2)
	created := 0
	for _, task := range tasks {
		if created >= cfg.Max {
			break
		}
		if len(cfg.Filter) > 0 && !cfg.Filter[task.ProjectID] {
			continue
		}
		key := task.ProjectID + ":" + task.Issue.ID
		_, acquired, err := cfg.LockMgr.TryAcquire(task.Issue.ID, key)
		if err != nil || !acquired {
			continue
		}
		proj, ok := cfg.Store.Get(task.ProjectID)
		if !ok {
			cfg.LockMgr.Release(task.Issue.ID)
			continue
		}
		run := buildAgentRun(&task, proj, cfg.Namespace)
		if err := cfg.K8s.Create(ctx, run); err != nil {
			log.Printf("create AgentRun %s: %v", run.Name, err)
			cfg.LockMgr.Release(task.Issue.ID)
			continue
		}
		created++
		log.Printf("created AgentRun %s for %s", run.Name, task.Issue.ID)
	}
}

func buildAgentRun(task *federation.FederatedTask, proj *registry.Project, namespace string) *v1alpha1.AgentRun {
	model := resolveModel(task.Issue.Labels)
	workstream := resolveWorkstream(task.Issue.Labels)
	repo := proj.RepoURL
	if proj.Fork && proj.UpstreamURL != "" {
		repo = proj.UpstreamURL
	}
	baseBranch := proj.RepoBranch
	if baseBranch == "" {
		baseBranch = "main"
	}
	name := "ar-" + task.ProjectID + "-" + strings.ReplaceAll(task.Issue.ID, ".", "-")
	return &v1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"beads.issue": task.Issue.ID, "project": task.ProjectID},
		},
		Spec: v1alpha1.AgentRunSpec{
			IssueID:    task.Issue.ID,
			Repo:       repo,
			BaseBranch: baseBranch,
			Model:      model,
			Workstream: workstream,
		},
		Status: v1alpha1.AgentRunStatus{Phase: ""},
	}
}

func resolveModel(labels []string) string {
	for _, l := range labels {
		if strings.HasPrefix(l, "model:") {
			return strings.TrimPrefix(l, "model:")
		}
	}
	return "glm-4.7"
}

func resolveWorkstream(labels []string) string {
	for _, l := range labels {
		if strings.HasPrefix(l, "workstream:") {
			return strings.TrimPrefix(l, "workstream:")
		}
	}
	return "builder"
}

func envDuration(key string, def time.Duration) time.Duration {
	if s := os.Getenv(key); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return def
}

func envInt(key string, def int) int {
	if s := os.Getenv(key); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return def
}

func envStr(key, def string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return def
}

func getKubeConfig() (*rest.Config, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if home := os.Getenv("HOME"); home != "" {
		path := home + "/.kube/config"
		if _, err := os.Stat(path); err == nil {
			return clientcmd.BuildConfigFromFlags("", path)
		}
	}
	return rest.InClusterConfig()
}

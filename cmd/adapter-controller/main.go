// adapter-controller watches Task/AgentRun CRDs and drives the adapter layer.
// Uses controller-runtime Manager with TaskReconciler.
package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/adapter"
	"sdp_dev/internal/agent"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/policy"
	"sdp_dev/internal/observability"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	shutdown, err := observability.SetupTracing("adapter-controller")
	if err != nil {
		setupLog.Info("OTLP tracing disabled", "reason", err)
	} else if shutdown != nil {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdown(ctx)
		}()
	}

	workDir, _ := os.Getwd()
	if d := os.Getenv("SDP_WORK_DIR"); d != "" {
		workDir = d
	}

	// Load model policy from ConfigMap when MODEL_POLICY_PATH set (WS-012-01)
	if policyPath := os.Getenv("MODEL_POLICY_PATH"); policyPath != "" {
		if _, err := policy.LoadFromPath(policyPath); err != nil {
			setupLog.Info("model policy load failed, using built-in allowlist", "path", policyPath, "err", err)
		} else {
			go policy.StartReloadWatcher(context.Background(), policyPath, 30*time.Second)
		}
	}

	cfg, err := getKubeConfig()
	if err != nil {
		setupLog.Error(err, "failed to get kube config")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: ":8081",
	})
	if err != nil {
		setupLog.Error(err, "failed to create manager")
		os.Exit(1)
	}

	// Adapter components for TaskReconciler.
	// SDP_LOCK_DIR: persistent path for run locks (default: os.TempDir); use a PVC path in K8s so locks survive restarts.
	lockDir := os.Getenv("SDP_LOCK_DIR")
	if lockDir == "" {
		lockDir = filepath.Join(os.TempDir(), "sdp-adapter-locks")
	}
	lockMgr := adapter.NewRunLockManager(lockDir)
	policyGate := adapter.NewPolicyGate()
	health := adapter.CheckWorkspaceHealth(workDir)
	if !health.BeadsAvailable {
		setupLog.Info("beads disabled", "reason", health.Reason, "workDir", workDir)
	}
	workspaceResolver := adapter.NewWorkspaceResolver(workDir)
	lifecycleReconciler := adapter.NewLifecycleReconciler()

	var traceEmitter *agent.TraceEmitter
	var natsBus bus.Bus
	if url := os.Getenv("NATS_URL"); url != "" {
		b, err := bus.ConnectAndProvision(context.Background(), url)
		if err == nil {
			defer b.Close()
			natsBus = b
			traceEmitter = agent.NewTraceEmitter(b, "adapter", "adapter-1", "adapter-controller", "reconciler", workDir)
		}
	}

	reconcilerOpts := adapter.TaskReconcilerOpts{
		WorkspaceResolver:   workspaceResolver,
		BeadsAvailable:      health.BeadsAvailable,
		LockManager:         lockMgr,
		PolicyGate:          policyGate,
		LifecycleReconciler: lifecycleReconciler,
		TraceEmitter:        traceEmitter,
		Bus:                 natsBus,
	}
	if err := adapter.NewTaskReconciler(mgr.GetClient(), mgr.GetScheme(), reconcilerOpts).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup TaskReconciler")
		os.Exit(1)
	}

	intentTranslator := adapter.NewIntentTranslator()
	healthChecker := policy.StubProviderHealthChecker{}
	agentRunOpts := adapter.AgentRunReconcilerOpts{
		IntentTranslator:      intentTranslator,
		PolicyGate:            policyGate,
		WorkspaceResolver:     workspaceResolver,
		BeadsAvailable:        health.BeadsAvailable,
		ProviderHealthChecker: healthChecker,
		Bus:                   natsBus,
	}
	if err := adapter.NewAgentRunReconciler(mgr.GetClient(), mgr.GetScheme(), agentRunOpts).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup AgentRunReconciler")
		os.Exit(1)
	}

	// Metrics endpoint (FR-017 / sdp_dev-rbl)
	metricsAddr := ":8082"
	if a := os.Getenv("METRICS_ADDR"); a != "" {
		metricsAddr = a
	}
	go func() {
		if err := observability.ServeMetrics(context.Background(), metricsAddr); err != nil {
			setupLog.Info("metrics server stopped", "err", err)
		}
	}()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	done := make(chan error, 1)
	go func() {
		done <- mgr.Start(ctx)
	}()

	<-ctx.Done()
	setupLog.Info("shutting down")
	if err := <-done; err != nil && err != context.Canceled {
		setupLog.Error(err, "manager shutdown error")
	}
}

func getKubeConfig() (*rest.Config, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if home := os.Getenv("HOME"); home != "" {
		path := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(path); err == nil {
			return clientcmd.BuildConfigFromFlags("", path)
		}
	}
	return rest.InClusterConfig()
}

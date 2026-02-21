// adapter-controller watches Task/AgentRun CRDs and drives the adapter layer.
// Uses controller-runtime Manager with TaskReconciler.
package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/adapter"
	"sdp_dev/internal/bus"

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

	cfg, err := getKubeConfig()
	if err != nil {
		setupLog.Error(err, "failed to get kube config")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		ctrl.Log.WithName("main").Error(err, "failed to create manager")
		os.Exit(1)
	}

	// Register TaskReconciler
	if err := adapter.NewTaskReconciler(mgr.GetClient(), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		ctrl.Log.WithName("main").Error(err, "failed to setup TaskReconciler")
		os.Exit(1)
	}

	// NATS connection (optional for status events)
	if url := os.Getenv("NATS_URL"); url != "" {
		natsClient := bus.NewClient(url)
		ctx, cancel := context.WithTimeout(context.Background(), bus.DefaultReconnectWait*5)
		if err := natsClient.Connect(ctx); err != nil {
			setupLog.Info("NATS connect skipped", "error", err)
		} else {
			defer natsClient.Close()
		}
		cancel()
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		if err := mgr.Start(ctx); err != nil {
			ctrl.Log.WithName("main").Error(err, "manager exited")
		}
	}()

	<-ctx.Done()
	setupLog.Info("shutting down")
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

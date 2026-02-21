package main

import (
	"context"
	"path/filepath"
	"testing"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/adapter"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestManagerStartsAndStops(t *testing.T) {
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "deploy", "k8s", "crd")},
		ErrorIfCRDPathMissing: false,
	}

	cfg, err := env.Start()
	if err != nil {
		t.Skipf("envtest start skipped (binaries may be missing): %v", err)
	}
	defer env.Stop()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	// Wire TaskReconciler with minimal opts (no beads, no NATS)
	reconcilerOpts := adapter.TaskReconcilerOpts{
		WorkDir:             t.TempDir(),
		LockManager:         adapter.NewRunLockManager(t.TempDir()),
		PolicyGate:          adapter.NewPolicyGate(),
		EvidenceProjector:   adapter.NewEvidenceProjector(t.TempDir()),
		LifecycleReconciler: adapter.NewLifecycleReconciler(),
	}
	if err := adapter.NewTaskReconciler(mgr.GetClient(), mgr.GetScheme(), reconcilerOpts).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup TaskReconciler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- mgr.Start(ctx)
	}()

	cancel()
	if err := <-done; err != nil && err != context.Canceled {
		t.Errorf("manager stopped with error: %v", err)
	}
}

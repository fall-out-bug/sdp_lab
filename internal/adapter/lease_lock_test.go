package adapter

import (
	"testing"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLeaseName(t *testing.T) {
	tests := []struct {
		issueID string
		want   string
	}{
		{"sdp_dev-4pg", "sdp-ar-sdp-dev-4pg"},
		{"sdp_dev-5l9.3", "sdp-ar-sdp-dev-5l9-3"},
		{"ABC_xyz", "sdp-ar-abc-xyz"},
		{"a", "sdp-ar-a"},
	}
	for _, tt := range tests {
		got := leaseName(tt.issueID)
		if got != tt.want {
			t.Errorf("leaseName(%q) = %q, want %q", tt.issueID, got, tt.want)
		}
		// DNS-1123: lowercase, alphanumeric, hyphens
		for _, r := range got {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				t.Errorf("leaseName(%q) produced invalid char %q in %q", tt.issueID, r, got)
			}
		}
		if len(got) > 63 {
			t.Errorf("leaseName(%q) = %q exceeds 63 chars", tt.issueID, got)
		}
	}
}

func TestLeaseLockManager_TryAcquire_InvalidIssueID(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	m := NewLeaseLockManager(c, "default")
	_, ok, err := m.TryAcquire("../../../etc", "run-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected invalid issueID to be rejected")
	}
}

func TestLeaseLockManager_Release_InvalidIssueID(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	m := NewLeaseLockManager(c, "default")
	err := m.Release("invalid..path")
	if err != nil {
		t.Fatalf("Release with invalid ID should return nil, got %v", err)
	}
}

func TestLeaseLockManager_TryAcquire_Release(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	m := NewLeaseLockManager(c, "default")

	runID, ok, err := m.TryAcquire("sdp_dev-4pg", "run-1")
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if !ok || runID != "run-1" {
		t.Errorf("expected acquired run-1, got ok=%v runID=%q", ok, runID)
	}

	_, ok2, _ := m.TryAcquire("sdp_dev-4pg", "run-2")
	if ok2 {
		t.Error("expected duplicate acquire to fail")
	}

	if err := m.Release("sdp_dev-4pg"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	runID3, ok3, err := m.TryAcquire("sdp_dev-4pg", "run-3")
	if err != nil {
		t.Fatalf("TryAcquire after Release: %v", err)
	}
	if !ok3 || runID3 != "run-3" {
		t.Errorf("expected re-acquire run-3, got ok=%v runID=%q", ok3, runID3)
	}
}

func TestLeaseLockManager_TryAcquire_AlreadyExists(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	holder := "other-run"
	existing := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "sdp-ar-sdp-dev-4pg", Namespace: "default"},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	m := NewLeaseLockManager(c, "default")
	_, ok, err := m.TryAcquire("sdp_dev-4pg", "run-1")
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if ok {
		t.Error("expected AlreadyExists to return false")
	}
}

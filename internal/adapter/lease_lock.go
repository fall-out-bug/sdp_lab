// Package adapter provides LeaseLockManager for distributed K8s-native locking.
package adapter

import (
	"context"
	"strings"

	"sdp_dev/internal/safeid"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ RunLock = (*LeaseLockManager)(nil)

// RunLock is the interface for issue-run locking (TryAcquire/Release).
type RunLock interface {
	TryAcquire(issueID, runID string) (string, bool, error)
	Release(issueID string) error
}

// LeaseLockManager uses K8s Lease for distributed locking across pods.
type LeaseLockManager struct {
	client    client.Client
	namespace string
}

// NewLeaseLockManager returns a LeaseLockManager for the given K8s client and namespace.
func NewLeaseLockManager(c client.Client, namespace string) *LeaseLockManager {
	return &LeaseLockManager{client: c, namespace: namespace}
}

// leaseName returns a DNS-1123 subdomain name for the issue.
// DNS-1123: lowercase, alphanumeric, hyphens; max 63 chars.
func leaseName(issueID string) string {
	s := strings.ToLower(issueID)
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "_", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	s = b.String()
	if len(s) > 50 {
		s = s[:50]
	}
	return "sdp-ar-" + s
}

// TryAcquire attempts to acquire a lock by creating a Lease. Returns (runID, true) if acquired.
func (m *LeaseLockManager) TryAcquire(issueID, runID string) (string, bool, error) {
	if err := safeid.ValidateIssueID(issueID); err != nil {
		return "", false, nil
	}
	ctx := context.Background()
	name := leaseName(issueID)
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.namespace,
			Labels:    map[string]string{"beads.issue": issueID, "run-id": runID},
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity: ptr(runID),
		},
	}
	err := m.client.Create(ctx, lease)
	if err != nil {
		if client.IgnoreAlreadyExists(err) == nil {
			return "", false, nil
		}
		return "", false, err
	}
	return runID, true, nil
}

// Release deletes the Lease for the issue.
func (m *LeaseLockManager) Release(issueID string) error {
	if err := safeid.ValidateIssueID(issueID); err != nil {
		return nil
	}
	ctx := context.Background()
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName(issueID),
			Namespace: m.namespace,
		},
	}
	return client.IgnoreNotFound(m.client.Delete(ctx, lease))
}

func ptr(s string) *string { return &s }

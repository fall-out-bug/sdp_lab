package modelgateway

import (
	"context"
	"testing"
	"time"
)

func TestCredentialManagerCreate(t *testing.T) {
	store := NewInMemoryCredentialStore()
	audit := NewInMemoryAuditLog()
	cm := NewCredentialManager(WithCredentialStore(store), WithAuditLogger(audit))

	cred, err := cm.CreateCredential(context.Background(), "tenant-1", "openai", "sk-test-key")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if cred.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", cred.TenantID)
	}
	if cred.ProviderID != "openai" {
		t.Errorf("expected openai, got %s", cred.ProviderID)
	}
	if cred.Status != CredentialStatusActive {
		t.Errorf("expected active status, got %s", cred.Status)
	}
}

func TestCredentialManagerCreateWithExpiry(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	expiry := time.Now().Add(30 * 24 * time.Hour)
	cred, err := cm.CreateCredential(context.Background(), "tenant-1", "openai", "sk-test",
		WithExpiry(expiry))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if cred.ExpiresAt == nil {
		t.Error("expected expiry to be set")
	}
}

func TestCredentialManagerGet(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	_, _ = cm.CreateCredential(context.Background(), "tenant-1", "openai", "sk-test")

	cred, err := cm.GetCredential(context.Background(), "tenant-1", "openai")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if cred.APIKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", cred.APIKey)
	}
}

func TestCredentialManagerGetNotFound(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	_, err := cm.GetCredential(context.Background(), "tenant-1", "openai")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestCredentialManagerGetExpired(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	past := time.Now().Add(-24 * time.Hour)
	cred, _ := cm.CreateCredential(context.Background(), "tenant-1", "openai", "sk-test",
		WithExpiry(past))

	cred.Status = CredentialStatusExpired
	_ = store.Set(context.Background(), cred)

	_, err := cm.GetCredential(context.Background(), "tenant-1", "openai")
	if err == nil {
		t.Error("expected error for expired credential")
	}
}

func TestCredentialManagerRotate(t *testing.T) {
	store := NewInMemoryCredentialStore()
	audit := NewInMemoryAuditLog()
	cm := NewCredentialManager(WithCredentialStore(store), WithAuditLogger(audit))

	_, _ = cm.CreateCredential(context.Background(), "tenant-1", "openai", "old-key")

	newCred, err := cm.RotateCredential(context.Background(), "tenant-1", "openai", "new-key")
	if err != nil {
		t.Fatalf("rotate failed: %v", err)
	}
	if newCred.APIKey != "new-key" {
		t.Errorf("expected new-key, got %s", newCred.APIKey)
	}

	storedCred, _ := store.Get(context.Background(), "tenant-1", "openai")
	if storedCred.Status != CredentialStatusActive {
		t.Errorf("expected active credential after rotation, got %s", storedCred.Status)
	}
	if storedCred.APIKey != "new-key" {
		t.Errorf("expected store to keep new key, got %s", storedCred.APIKey)
	}
}

func TestCredentialManagerRevoke(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	_, _ = cm.CreateCredential(context.Background(), "tenant-1", "openai", "sk-test")

	err := cm.RevokeCredential(context.Background(), "tenant-1", "openai")
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	cred, _ := store.Get(context.Background(), "tenant-1", "openai")
	if cred.Status != CredentialStatusRevoked {
		t.Error("expected revoked status")
	}
	if cred.APIKey != "" {
		t.Error("expected API key to be cleared")
	}
}

func TestCredentialManagerCheckExpiry(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	expiring := time.Now().Add(3 * 24 * time.Hour)
	_, _ = cm.CreateCredential(context.Background(), "tenant-1", "openai", "sk-1",
		WithExpiry(expiring))

	notExpiring := time.Now().Add(30 * 24 * time.Hour)
	_, _ = cm.CreateCredential(context.Background(), "tenant-1", "anthropic", "sk-2",
		WithExpiry(notExpiring))

	expiringCreds, err := cm.CheckExpiry(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("check expiry failed: %v", err)
	}
	if len(expiringCreds) != 1 {
		t.Errorf("expected 1 expiring credential, got %d", len(expiringCreds))
	}
}

func TestCredentialManagerRotationStatus(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	cred, _ := cm.CreateCredential(context.Background(), "tenant-1", "openai", "old-key")
	_, _ = cm.RotateCredential(context.Background(), "tenant-1", "openai", "new-key")

	state, ok := cm.GetRotationStatus(cred.ID)
	if !ok {
		t.Fatal("expected rotation state")
	}
	if state.Status != "completed" {
		t.Errorf("expected completed status, got %s", state.Status)
	}
}

func TestAuditLog(t *testing.T) {
	store := NewInMemoryCredentialStore()
	audit := NewInMemoryAuditLog()
	cm := NewCredentialManager(WithCredentialStore(store), WithAuditLogger(audit))

	_, _ = cm.CreateCredential(context.Background(), "tenant-1", "openai", "sk-test")
	_, _ = cm.GetCredential(context.Background(), "tenant-1", "openai")

	entries, err := audit.Query(context.Background(), "tenant-1", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(entries) < 2 {
		t.Errorf("expected at least 2 audit entries, got %d", len(entries))
	}
}

func TestInMemoryCredentialStore(t *testing.T) {
	store := NewInMemoryCredentialStore()

	cred := &Credential{
		ID:         "cred-1",
		TenantID:   "tenant-1",
		ProviderID: "openai",
		APIKey:     "sk-test",
		Status:     CredentialStatusActive,
	}

	err := store.Set(context.Background(), cred)
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	got, err := store.Get(context.Background(), "tenant-1", "openai")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.APIKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", got.APIKey)
	}

	err = store.Delete(context.Background(), "tenant-1", "openai")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = store.Get(context.Background(), "tenant-1", "openai")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestInMemoryCredentialStoreList(t *testing.T) {
	store := NewInMemoryCredentialStore()

	store.Set(context.Background(), &Credential{TenantID: "tenant-1", ProviderID: "openai"})
	store.Set(context.Background(), &Credential{TenantID: "tenant-1", ProviderID: "anthropic"})
	store.Set(context.Background(), &Credential{TenantID: "tenant-2", ProviderID: "openai"})

	creds, err := store.List(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(creds) != 2 {
		t.Errorf("expected 2 credentials for tenant-1, got %d", len(creds))
	}
}

func TestInMemoryAuditLog(t *testing.T) {
	audit := NewInMemoryAuditLog()

	entry := AuditLogEntry{
		ID:         "entry-1",
		Timestamp:  time.Now(),
		TenantID:   "tenant-1",
		ProviderID: "openai",
		Action:     "get",
		Actor:      "system",
		Success:    true,
	}

	err := audit.Log(context.Background(), entry)
	if err != nil {
		t.Fatalf("log failed: %v", err)
	}

	entries, err := audit.Query(context.Background(), "tenant-1", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestCredentialIsolatedPerTenant(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	_, _ = cm.CreateCredential(context.Background(), "tenant-1", "openai", "sk-tenant1")
	_, _ = cm.CreateCredential(context.Background(), "tenant-2", "openai", "sk-tenant2")

	cred1, _ := cm.GetCredential(context.Background(), "tenant-1", "openai")
	cred2, _ := cm.GetCredential(context.Background(), "tenant-2", "openai")

	if cred1.APIKey == cred2.APIKey {
		t.Error("credentials should be isolated per tenant")
	}
}

func TestCredentialNoStore(t *testing.T) {
	cm := NewCredentialManager()

	_, err := cm.GetCredential(context.Background(), "tenant-1", "openai")
	if err == nil {
		t.Error("expected error when no store configured")
	}
}

func TestCredentialStatus(t *testing.T) {
	statuses := []CredentialStatus{
		CredentialStatusActive,
		CredentialStatusExpired,
		CredentialStatusRotating,
		CredentialStatusRevoked,
	}

	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("unexpected empty status")
		}
	}
}

func TestCredentialWithBaseURL(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	cred, err := cm.CreateCredential(context.Background(), "tenant-1", "selfhosted", "key",
		WithBaseURL("https://models.internal.company.com"))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if cred.BaseURL != "https://models.internal.company.com" {
		t.Errorf("expected base URL, got %s", cred.BaseURL)
	}
}

func TestCredentialWithRotationDue(t *testing.T) {
	store := NewInMemoryCredentialStore()
	cm := NewCredentialManager(WithCredentialStore(store))

	rotationDue := time.Now().Add(90 * 24 * time.Hour)
	cred, err := cm.CreateCredential(context.Background(), "tenant-1", "openai", "key",
		WithRotationDue(rotationDue))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if cred.RotationDue == nil {
		t.Error("expected rotation due to be set")
	}
}

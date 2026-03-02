package modelgateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type CredentialID string

type CredentialStatus string

const (
	CredentialStatusActive   CredentialStatus = "active"
	CredentialStatusExpired  CredentialStatus = "expired"
	CredentialStatusRotating CredentialStatus = "rotating"
	CredentialStatusRevoked  CredentialStatus = "revoked"
)

type Credential struct {
	ID          CredentialID           `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	ProviderID  ProviderID             `json:"provider_id"`
	APIKey      string                 `json:"api_key,omitempty"`
	BaseURL     string                 `json:"base_url,omitempty"`
	Status      CredentialStatus       `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time             `json:"last_used_at,omitempty"`
	RotationDue *time.Time             `json:"rotation_due,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CredentialStore interface {
	Get(ctx context.Context, tenantID string, providerID ProviderID) (*Credential, error)
	Set(ctx context.Context, cred *Credential) error
	Delete(ctx context.Context, tenantID string, providerID ProviderID) error
	List(ctx context.Context, tenantID string) ([]*Credential, error)
}

type AuditLogEntry struct {
	ID         string     `json:"id"`
	Timestamp  time.Time  `json:"timestamp"`
	TenantID   string     `json:"tenant_id"`
	ProviderID ProviderID `json:"provider_id"`
	Action     string     `json:"action"`
	Actor      string     `json:"actor"`
	Success    bool       `json:"success"`
	Error      string     `json:"error,omitempty"`
}

type AuditLogger interface {
	Log(ctx context.Context, entry AuditLogEntry) error
	Query(ctx context.Context, tenantID string, since time.Time) ([]AuditLogEntry, error)
}

type CredentialManager struct {
	mu        sync.RWMutex
	store     CredentialStore
	audit     AuditLogger
	rotations map[CredentialID]*RotationState
}

type RotationState struct {
	CredentialID CredentialID
	OldKey       string
	NewKey       string
	StartedAt    time.Time
	CompletedAt  *time.Time
	Status       string
}

type CredentialManagerOption func(*CredentialManager)

func WithCredentialStore(store CredentialStore) CredentialManagerOption {
	return func(cm *CredentialManager) {
		cm.store = store
	}
}

func WithAuditLogger(audit AuditLogger) CredentialManagerOption {
	return func(cm *CredentialManager) {
		cm.audit = audit
	}
}

func NewCredentialManager(opts ...CredentialManagerOption) *CredentialManager {
	cm := &CredentialManager{
		rotations: make(map[CredentialID]*RotationState),
	}
	for _, opt := range opts {
		opt(cm)
	}
	return cm
}

func (cm *CredentialManager) GetCredential(ctx context.Context, tenantID string, providerID ProviderID) (*Credential, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.store == nil {
		return nil, fmt.Errorf("credential store not configured")
	}

	cred, err := cm.store.Get(ctx, tenantID, providerID)
	if err != nil {
		cm.auditLog(ctx, tenantID, providerID, "get", "system", false, err.Error())
		return nil, err
	}

	if cred.Status == CredentialStatusExpired {
		cm.auditLog(ctx, tenantID, providerID, "get", "system", false, "credential expired")
		return nil, fmt.Errorf("credential expired")
	}

	if cred.Status == CredentialStatusRevoked {
		cm.auditLog(ctx, tenantID, providerID, "get", "system", false, "credential revoked")
		return nil, fmt.Errorf("credential revoked")
	}

	now := time.Now()
	cred.LastUsedAt = &now
	_ = cm.store.Set(ctx, cred)

	cm.auditLog(ctx, tenantID, providerID, "get", "system", true, "")
	return cred, nil
}

func (cm *CredentialManager) CreateCredential(ctx context.Context, tenantID string, providerID ProviderID, apiKey string, opts ...CredentialOption) (*Credential, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cred := &Credential{
		ID:         CredentialID(generateID()),
		TenantID:   tenantID,
		ProviderID: providerID,
		APIKey:     apiKey,
		Status:     CredentialStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(cred)
	}

	if cm.store != nil {
		if err := cm.store.Set(ctx, cred); err != nil {
			return nil, err
		}
	}

	cm.auditLog(ctx, tenantID, providerID, "create", "system", true, "")
	return cred, nil
}

type CredentialOption func(*Credential)

func WithExpiry(expiresAt time.Time) CredentialOption {
	return func(c *Credential) {
		c.ExpiresAt = &expiresAt
	}
}

func WithBaseURL(baseURL string) CredentialOption {
	return func(c *Credential) {
		c.BaseURL = baseURL
	}
}

func WithRotationDue(rotationDue time.Time) CredentialOption {
	return func(c *Credential) {
		c.RotationDue = &rotationDue
	}
}

func (cm *CredentialManager) RotateCredential(ctx context.Context, tenantID string, providerID ProviderID, newAPIKey string) (*Credential, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.store == nil {
		return nil, fmt.Errorf("credential store not configured")
	}

	oldCred, err := cm.store.Get(ctx, tenantID, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing credential: %w", err)
	}

	rotationState := &RotationState{
		CredentialID: oldCred.ID,
		OldKey:       oldCred.APIKey,
		NewKey:       newAPIKey,
		StartedAt:    time.Now(),
		Status:       "in_progress",
	}
	cm.rotations[oldCred.ID] = rotationState

	oldCred.Status = CredentialStatusRotating
	_ = cm.store.Set(ctx, oldCred)

	newCred := &Credential{
		ID:         CredentialID(generateID()),
		TenantID:   tenantID,
		ProviderID: providerID,
		APIKey:     newAPIKey,
		Status:     CredentialStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := cm.store.Set(ctx, newCred); err != nil {
		rotationState.Status = "failed"
		return nil, fmt.Errorf("failed to store new credential: %w", err)
	}

	now := time.Now()
	rotationState.CompletedAt = &now
	rotationState.Status = "completed"

	oldCred.Status = CredentialStatusRevoked
	_ = cm.store.Set(ctx, oldCred)

	cm.auditLog(ctx, tenantID, providerID, "rotate", "system", true, "")
	return newCred, nil
}

func (cm *CredentialManager) RevokeCredential(ctx context.Context, tenantID string, providerID ProviderID) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.store == nil {
		return fmt.Errorf("credential store not configured")
	}

	cred, err := cm.store.Get(ctx, tenantID, providerID)
	if err != nil {
		return err
	}

	cred.Status = CredentialStatusRevoked
	cred.APIKey = ""
	cred.UpdatedAt = time.Now()

	if err := cm.store.Set(ctx, cred); err != nil {
		return err
	}

	cm.auditLog(ctx, tenantID, providerID, "revoke", "system", true, "")
	return nil
}

func (cm *CredentialManager) CheckExpiry(ctx context.Context, tenantID string) ([]*Credential, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.store == nil {
		return nil, fmt.Errorf("credential store not configured")
	}

	creds, err := cm.store.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var expiring []*Credential
	now := time.Now()
	alertWindow := 7 * 24 * time.Hour

	for _, cred := range creds {
		if cred.ExpiresAt != nil {
			timeUntilExpiry := cred.ExpiresAt.Sub(now)
			if timeUntilExpiry <= alertWindow && cred.Status == CredentialStatusActive {
				expiring = append(expiring, cred)
			}
			if cred.ExpiresAt.Before(now) && cred.Status == CredentialStatusActive {
				cred.Status = CredentialStatusExpired
				_ = cm.store.Set(ctx, cred)
			}
		}
	}

	return expiring, nil
}

func (cm *CredentialManager) GetRotationStatus(credID CredentialID) (*RotationState, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	state, ok := cm.rotations[credID]
	return state, ok
}

func (cm *CredentialManager) auditLog(ctx context.Context, tenantID string, providerID ProviderID, action, actor string, success bool, errMsg string) {
	if cm.audit == nil {
		return
	}

	entry := AuditLogEntry{
		ID:         generateID(),
		Timestamp:  time.Now(),
		TenantID:   tenantID,
		ProviderID: providerID,
		Action:     action,
		Actor:      actor,
		Success:    success,
		Error:      errMsg,
	}
	_ = cm.audit.Log(ctx, entry)
}

type InMemoryCredentialStore struct {
	mu          sync.RWMutex
	credentials map[string]*Credential
}

func NewInMemoryCredentialStore() *InMemoryCredentialStore {
	return &InMemoryCredentialStore{
		credentials: make(map[string]*Credential),
	}
}

func (s *InMemoryCredentialStore) key(tenantID string, providerID ProviderID) string {
	return fmt.Sprintf("%s:%s", tenantID, providerID)
}

func (s *InMemoryCredentialStore) Get(ctx context.Context, tenantID string, providerID ProviderID) (*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.key(tenantID, providerID)
	cred, ok := s.credentials[key]
	if !ok {
		return nil, fmt.Errorf("credential not found for tenant %s, provider %s", tenantID, providerID)
	}
	return cred, nil
}

func (s *InMemoryCredentialStore) Set(ctx context.Context, cred *Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.key(cred.TenantID, cred.ProviderID)
	s.credentials[key] = cred
	return nil
}

func (s *InMemoryCredentialStore) Delete(ctx context.Context, tenantID string, providerID ProviderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.key(tenantID, providerID)
	delete(s.credentials, key)
	return nil
}

func (s *InMemoryCredentialStore) List(ctx context.Context, tenantID string) ([]*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var creds []*Credential
	for k, cred := range s.credentials {
		if len(k) > len(tenantID) && k[:len(tenantID)] == tenantID {
			creds = append(creds, cred)
		}
	}
	return creds, nil
}

type InMemoryAuditLog struct {
	mu      sync.RWMutex
	entries []AuditLogEntry
}

func NewInMemoryAuditLog() *InMemoryAuditLog {
	return &InMemoryAuditLog{
		entries: make([]AuditLogEntry, 0),
	}
}

func (l *InMemoryAuditLog) Log(ctx context.Context, entry AuditLogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, entry)
	return nil
}

func (l *InMemoryAuditLog) Query(ctx context.Context, tenantID string, since time.Time) ([]AuditLogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []AuditLogEntry
	for _, entry := range l.entries {
		if entry.TenantID == tenantID && entry.Timestamp.After(since) {
			results = append(results, entry)
		}
	}
	return results, nil
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

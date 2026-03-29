package authz

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TenantID string

type Role string

const (
	RoleOperator Role = "operator"
	RoleReviewer Role = "reviewer"
	RoleAuditor  Role = "auditor"
	RoleAdmin    Role = "admin"
)

type Permission string

const (
	PermissionReadWorkstream    Permission = "workstream:read"
	PermissionWriteWorkstream   Permission = "workstream:write"
	PermissionDeleteWorkstream  Permission = "workstream:delete"
	PermissionReadEvidence      Permission = "evidence:read"
	PermissionWriteEvidence     Permission = "evidence:write"
	PermissionReadAudit         Permission = "audit:read"
	PermissionManageTenants     Permission = "tenant:manage"
	PermissionManageCredentials Permission = "credentials:manage"
	PermissionExecuteTask       Permission = "task:execute"
	PermissionReviewTask        Permission = "task:review"
)

type TenantScope struct {
	TenantID   TenantID               `json:"tenant_id"`
	Name       string                 `json:"name"`
	Roles      []Role                 `json:"roles"`
	Namespaces []string               `json:"namespaces"`
	CreatedAt  time.Time              `json:"created_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type Subject struct {
	ID         string   `json:"id"`
	TenantID   TenantID `json:"tenant_id"`
	Roles      []Role   `json:"roles"`
	Namespaces []string `json:"namespaces"`
}

type AccessRequest struct {
	Subject        Subject                `json:"subject"`
	Resource       string                 `json:"resource"`
	Action         Permission             `json:"action"`
	ResourceTenant TenantID               `json:"resource_tenant,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}

type AccessDecision struct {
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason"`
	DeniedBy  string    `json:"denied_by,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type CrossTenantAccessLog struct {
	ID            string     `json:"id"`
	Timestamp     time.Time  `json:"timestamp"`
	SubjectID     string     `json:"subject_id"`
	SubjectTenant TenantID   `json:"subject_tenant"`
	TargetTenant  TenantID   `json:"target_tenant"`
	Resource      string     `json:"resource"`
	Action        Permission `json:"action"`
	Denied        bool       `json:"denied"`
}

var rolePermissions = map[Role][]Permission{
	RoleOperator: {
		PermissionReadWorkstream,
		PermissionWriteWorkstream,
		PermissionReadEvidence,
		PermissionWriteEvidence,
		PermissionExecuteTask,
		PermissionManageCredentials,
	},
	RoleReviewer: {
		PermissionReadWorkstream,
		PermissionReadEvidence,
		PermissionReviewTask,
	},
	RoleAuditor: {
		PermissionReadWorkstream,
		PermissionReadEvidence,
		PermissionReadAudit,
	},
	RoleAdmin: {
		PermissionReadWorkstream,
		PermissionWriteWorkstream,
		PermissionDeleteWorkstream,
		PermissionReadEvidence,
		PermissionWriteEvidence,
		PermissionReadAudit,
		PermissionManageTenants,
		PermissionManageCredentials,
		PermissionExecuteTask,
		PermissionReviewTask,
	},
}

type tenantAuthorizer struct {
	mu                sync.RWMutex
	scopes            map[TenantID]*TenantScope
	crossTenantLogger CrossTenantLogger
}

type CrossTenantLogger interface {
	Log(entry CrossTenantAccessLog) error
}

type tenantAuthorizerOption func(*tenantAuthorizer)

func withCrossTenantLogger(logger CrossTenantLogger) tenantAuthorizerOption {
	return func(a *tenantAuthorizer) {
		a.crossTenantLogger = logger
	}
}

func newTenantAuthorizer(opts ...tenantAuthorizerOption) *tenantAuthorizer {
	a := &tenantAuthorizer{
		scopes: make(map[TenantID]*TenantScope),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *tenantAuthorizer) RegisterTenant(scope *TenantScope) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if scope.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}

	scope.CreatedAt = time.Now()
	a.scopes[scope.TenantID] = scope
	return nil
}

func (a *tenantAuthorizer) GetTenant(id TenantID) (*TenantScope, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	scope, ok := a.scopes[id]
	return scope, ok
}

func (a *tenantAuthorizer) Authorize(ctx context.Context, req AccessRequest) AccessDecision {
	a.mu.RLock()
	defer a.mu.RUnlock()

	now := time.Now()

	if req.ResourceTenant != "" && req.Subject.TenantID != req.ResourceTenant {
		a.logCrossTenantAccess(req, true)
		return AccessDecision{
			Allowed:   false,
			Reason:    "cross-tenant access denied",
			DeniedBy:  "tenant_boundary",
			Timestamp: now,
		}
	}

	permissions := a.getPermissions(req.Subject.Roles)
	if !hasPermission(permissions, req.Action) {
		return AccessDecision{
			Allowed:   false,
			Reason:    fmt.Sprintf("permission %s not granted", req.Action),
			DeniedBy:  "rbac",
			Timestamp: now,
		}
	}

	return AccessDecision{
		Allowed:   true,
		Reason:    "access granted",
		Timestamp: now,
	}
}

func (a *tenantAuthorizer) getPermissions(roles []Role) []Permission {
	var perms []Permission
	seen := make(map[Permission]bool)

	for _, role := range roles {
		if rolePerms, ok := rolePermissions[role]; ok {
			for _, p := range rolePerms {
				if !seen[p] {
					seen[p] = true
					perms = append(perms, p)
				}
			}
		}
	}
	return perms
}

func (a *tenantAuthorizer) logCrossTenantAccess(req AccessRequest, denied bool) {
	if a.crossTenantLogger == nil {
		return
	}

	entry := CrossTenantAccessLog{
		ID:            generateID(),
		Timestamp:     time.Now(),
		SubjectID:     req.Subject.ID,
		SubjectTenant: req.Subject.TenantID,
		TargetTenant:  req.ResourceTenant,
		Resource:      req.Resource,
		Action:        req.Action,
		Denied:        denied,
	}
	_ = a.crossTenantLogger.Log(entry)
}

func (a *tenantAuthorizer) CheckNamespaceAccess(subject Subject, namespace string) bool {
	if len(subject.Namespaces) == 0 {
		return true
	}

	for _, ns := range subject.Namespaces {
		if ns == namespace || ns == "*" {
			return true
		}
	}
	return false
}

func (a *tenantAuthorizer) ListTenants() []TenantID {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var ids []TenantID
	for id := range a.scopes {
		ids = append(ids, id)
	}
	return ids
}

func hasPermission(perms []Permission, target Permission) bool {
	for _, p := range perms {
		if p == target {
			return true
		}
	}
	return false
}

type inMemoryCrossTenantLogger struct {
	mu      sync.RWMutex
	entries []CrossTenantAccessLog
}

func newInMemoryCrossTenantLogger() *inMemoryCrossTenantLogger {
	return &inMemoryCrossTenantLogger{
		entries: make([]CrossTenantAccessLog, 0),
	}
}

func (l *inMemoryCrossTenantLogger) Log(entry CrossTenantAccessLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, entry)
	return nil
}

func (l *inMemoryCrossTenantLogger) GetEntries() []CrossTenantAccessLog {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries := make([]CrossTenantAccessLog, len(l.entries))
	copy(entries, l.entries)
	return entries
}

type tenantMiddleware struct {
	authorizer *tenantAuthorizer
}

func newTenantMiddleware(authorizer *tenantAuthorizer) *tenantMiddleware {
	return &tenantMiddleware{authorizer: authorizer}
}

func (m *tenantMiddleware) ValidateTenantAccess(ctx context.Context, subject Subject, resourceTenant TenantID) error {
	if subject.TenantID != resourceTenant {
		return fmt.Errorf("tenant boundary violation: subject %s cannot access tenant %s", subject.TenantID, resourceTenant)
	}
	return nil
}

func (m *tenantMiddleware) RequirePermission(ctx context.Context, subject Subject, perm Permission) error {
	permissions := m.authorizer.getPermissions(subject.Roles)
	if !hasPermission(permissions, perm) {
		return fmt.Errorf("permission denied: %s required", perm)
	}
	return nil
}

func getRolePermissions(role Role) []Permission {
	if perms, ok := rolePermissions[role]; ok {
		return perms
	}
	return nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

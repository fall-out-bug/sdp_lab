package authz

import (
	"context"
	"testing"
)

func TestNewTenantAuthorizer(t *testing.T) {
	a := newTenantAuthorizer()
	if a == nil {
		t.Fatal("expected non-nil authorizer")
	}
}

func TestRegisterTenant(t *testing.T) {
	a := newTenantAuthorizer()

	scope := &TenantScope{
		TenantID: "tenant-1",
		Name:     "Test Tenant",
		Roles:    []Role{RoleOperator},
	}

	err := a.RegisterTenant(scope)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	got, ok := a.GetTenant("tenant-1")
	if !ok {
		t.Fatal("expected tenant to be found")
	}
	if got.Name != "Test Tenant" {
		t.Errorf("expected Test Tenant, got %s", got.Name)
	}
}

func TestRegisterTenantEmptyID(t *testing.T) {
	a := newTenantAuthorizer()

	scope := &TenantScope{
		TenantID: "",
		Name:     "Invalid",
	}

	err := a.RegisterTenant(scope)
	if err == nil {
		t.Error("expected error for empty tenant ID")
	}
}

func TestAuthorizeSameTenant(t *testing.T) {
	a := newTenantAuthorizer()

	req := AccessRequest{
		Subject: Subject{
			ID:       "user-1",
			TenantID: "tenant-1",
			Roles:    []Role{RoleOperator},
		},
		Resource:       "workstream-1",
		Action:         PermissionReadWorkstream,
		ResourceTenant: "tenant-1",
	}

	decision := a.Authorize(context.Background(), req)
	if !decision.Allowed {
		t.Errorf("expected allowed, got: %s", decision.Reason)
	}
}

func TestAuthorizeCrossTenantDenied(t *testing.T) {
	logger := newInMemoryCrossTenantLogger()
	a := newTenantAuthorizer(withCrossTenantLogger(logger))

	req := AccessRequest{
		Subject: Subject{
			ID:       "user-1",
			TenantID: "tenant-1",
			Roles:    []Role{RoleOperator},
		},
		Resource:       "workstream-1",
		Action:         PermissionReadWorkstream,
		ResourceTenant: "tenant-2",
	}

	decision := a.Authorize(context.Background(), req)
	if decision.Allowed {
		t.Error("expected denied for cross-tenant access")
	}
	if decision.DeniedBy != "tenant_boundary" {
		t.Errorf("expected denied by tenant_boundary, got %s", decision.DeniedBy)
	}

	entries := logger.GetEntries()
	if len(entries) != 1 {
		t.Errorf("expected 1 cross-tenant log entry, got %d", len(entries))
	}
}

func TestAuthorizePermissionDenied(t *testing.T) {
	a := newTenantAuthorizer()

	req := AccessRequest{
		Subject: Subject{
			ID:       "user-1",
			TenantID: "tenant-1",
			Roles:    []Role{RoleAuditor},
		},
		Resource:       "workstream-1",
		Action:         PermissionWriteWorkstream,
		ResourceTenant: "tenant-1",
	}

	decision := a.Authorize(context.Background(), req)
	if decision.Allowed {
		t.Error("expected denied for auditor writing workstream")
	}
	if decision.DeniedBy != "rbac" {
		t.Errorf("expected denied by rbac, got %s", decision.DeniedBy)
	}
}

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		role       Role
		permission Permission
		allowed    bool
	}{
		{RoleOperator, PermissionReadWorkstream, true},
		{RoleOperator, PermissionWriteWorkstream, true},
		{RoleOperator, PermissionManageTenants, false},
		{RoleReviewer, PermissionReadWorkstream, true},
		{RoleReviewer, PermissionWriteWorkstream, false},
		{RoleReviewer, PermissionReviewTask, true},
		{RoleAuditor, PermissionReadAudit, true},
		{RoleAuditor, PermissionWriteWorkstream, false},
		{RoleAdmin, PermissionManageTenants, true},
		{RoleAdmin, PermissionDeleteWorkstream, true},
	}

	a := newTenantAuthorizer()

	for _, tt := range tests {
		req := AccessRequest{
			Subject: Subject{
				ID:       "user-1",
				TenantID: "tenant-1",
				Roles:    []Role{tt.role},
			},
			Action:         tt.permission,
			ResourceTenant: "tenant-1",
		}

		decision := a.Authorize(context.Background(), req)
		if decision.Allowed != tt.allowed {
			t.Errorf("role %s permission %s: expected %v, got %v (reason: %s)",
				tt.role, tt.permission, tt.allowed, decision.Allowed, decision.Reason)
		}
	}
}

func TestMultipleRoles(t *testing.T) {
	a := newTenantAuthorizer()

	req := AccessRequest{
		Subject: Subject{
			ID:       "user-1",
			TenantID: "tenant-1",
			Roles:    []Role{RoleAuditor, RoleReviewer},
		},
		Action:         PermissionReviewTask,
		ResourceTenant: "tenant-1",
	}

	decision := a.Authorize(context.Background(), req)
	if !decision.Allowed {
		t.Errorf("expected allowed with multiple roles, got: %s", decision.Reason)
	}
}

func TestCheckNamespaceAccess(t *testing.T) {
	a := newTenantAuthorizer()

	tests := []struct {
		subject   Subject
		namespace string
		allowed   bool
	}{
		{
			subject:   Subject{Namespaces: []string{"ns-1"}},
			namespace: "ns-1",
			allowed:   true,
		},
		{
			subject:   Subject{Namespaces: []string{"ns-1"}},
			namespace: "ns-2",
			allowed:   false,
		},
		{
			subject:   Subject{Namespaces: []string{"*"}},
			namespace: "any-ns",
			allowed:   true,
		},
		{
			subject:   Subject{Namespaces: []string{}},
			namespace: "any-ns",
			allowed:   true,
		},
	}

	for _, tt := range tests {
		got := a.CheckNamespaceAccess(tt.subject, tt.namespace)
		if got != tt.allowed {
			t.Errorf("namespace %s: expected %v, got %v", tt.namespace, tt.allowed, got)
		}
	}
}

func TestListTenants(t *testing.T) {
	a := newTenantAuthorizer()

	_ = a.RegisterTenant(&TenantScope{TenantID: "tenant-1"})
	_ = a.RegisterTenant(&TenantScope{TenantID: "tenant-2"})

	tenants := a.ListTenants()
	if len(tenants) != 2 {
		t.Errorf("expected 2 tenants, got %d", len(tenants))
	}
}

func TestInMemoryCrossTenantLogger(t *testing.T) {
	logger := newInMemoryCrossTenantLogger()

	entry := CrossTenantAccessLog{
		ID:            "log-1",
		SubjectID:     "user-1",
		SubjectTenant: "tenant-1",
		TargetTenant:  "tenant-2",
		Resource:      "workstream-1",
		Action:        PermissionReadWorkstream,
		Denied:        true,
	}

	err := logger.Log(entry)
	if err != nil {
		t.Fatalf("log failed: %v", err)
	}

	entries := logger.GetEntries()
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestTenantMiddleware(t *testing.T) {
	a := newTenantAuthorizer()
	m := newTenantMiddleware(a)

	subject := Subject{
		ID:       "user-1",
		TenantID: "tenant-1",
	}

	err := m.ValidateTenantAccess(context.Background(), subject, "tenant-1")
	if err != nil {
		t.Errorf("expected no error for same tenant: %v", err)
	}

	err = m.ValidateTenantAccess(context.Background(), subject, "tenant-2")
	if err == nil {
		t.Error("expected error for cross-tenant access")
	}
}

func TestTenantMiddlewareRequirePermission(t *testing.T) {
	a := newTenantAuthorizer()
	m := newTenantMiddleware(a)

	operator := Subject{Roles: []Role{RoleOperator}}
	auditor := Subject{Roles: []Role{RoleAuditor}}

	err := m.RequirePermission(context.Background(), operator, PermissionWriteWorkstream)
	if err != nil {
		t.Errorf("operator should have write permission: %v", err)
	}

	err = m.RequirePermission(context.Background(), auditor, PermissionWriteWorkstream)
	if err == nil {
		t.Error("auditor should not have write permission")
	}
}

func TestGetRolePermissions(t *testing.T) {
	perms := getRolePermissions(RoleOperator)
	if len(perms) == 0 {
		t.Error("expected permissions for operator role")
	}

	perms = getRolePermissions(Role("nonexistent"))
	if perms != nil {
		t.Error("expected nil for nonexistent role")
	}
}

func TestAccessDecisionTimestamp(t *testing.T) {
	a := newTenantAuthorizer()

	req := AccessRequest{
		Subject: Subject{
			TenantID: "tenant-1",
			Roles:    []Role{RoleOperator},
		},
		ResourceTenant: "tenant-1",
		Action:         PermissionReadWorkstream,
	}

	decision := a.Authorize(context.Background(), req)
	if decision.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestTenantScopeMetadata(t *testing.T) {
	a := newTenantAuthorizer()

	scope := &TenantScope{
		TenantID: "tenant-1",
		Metadata: map[string]interface{}{
			"region": "us-east-1",
			"tier":   "enterprise",
		},
	}

	err := a.RegisterTenant(scope)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	got, _ := a.GetTenant("tenant-1")
	if got.Metadata["tier"] != "enterprise" {
		t.Error("expected metadata to be preserved")
	}
}

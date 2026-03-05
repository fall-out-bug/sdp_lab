# BYOM Credential Management

This document describes the credential management system for the Bring Your Own Model (BYOM) gateway.

## Overview

The credential management system provides:
- Tenant-scoped credential isolation
- Secure credential rotation
- Audit logging for all credential access
- Key expiry monitoring and alerts

## Architecture

```
┌─────────────────┐     ┌───────────────────┐     ┌─────────────────┐
│   Tenant App    │────▶│  CredentialManager │────▶│ CredentialStore │
└─────────────────┘     └───────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌──────────────┐
                        │  AuditLog    │
                        └──────────────┘
```

## Credential Lifecycle

### States

| State | Description |
|-------|-------------|
| `active` | Credential is in use |
| `expired` | Credential past expiry date |
| `rotating` | Rotation in progress |
| `revoked` | Credential no longer valid |

### Creating Credentials

```go
cm := NewCredentialManager(
    WithCredentialStore(store),
    WithAuditLogger(audit),
)

cred, err := cm.CreateCredential(ctx, "tenant-1", "openai", "sk-xxx",
    WithExpiry(time.Now().Add(90 * 24 * time.Hour)),
    WithRotationDue(time.Now().Add(60 * 24 * time.Hour)),
)
```

### Retrieving Credentials

```go
cred, err := cm.GetCredential(ctx, "tenant-1", "openai")
if err != nil {
    // Handle error (not found, expired, revoked)
}
// Use cred.APIKey for provider requests
```

## Credential Rotation

### Automatic Rotation

Check for credentials needing rotation:

```go
expiring, err := cm.CheckExpiry(ctx, tenantID)
for _, cred := range expiring {
    // Alert or trigger rotation workflow
}
```

### Manual Rotation

```go
newCred, err := cm.RotateCredential(ctx, "tenant-1", "openai", "sk-new-key")
```

Rotation process:
1. Old credential marked as `rotating`
2. New credential created with `active` status
3. Old credential revoked after successful creation
4. Audit log records the rotation event

### Rotation Runbook

1. **Prepare new key**: Obtain new API key from provider
2. **Trigger rotation**: Call `RotateCredential` with new key
3. **Verify**: Confirm new credential is active
4. **Monitor**: Watch for any authentication errors
5. **Rollback** (if needed): Use old key from rotation state

## Audit Logging

All credential operations are logged:

| Action | Description |
|--------|-------------|
| `create` | New credential created |
| `get` | Credential accessed |
| `rotate` | Credential rotated |
| `revoke` | Credential revoked |

Query audit log:

```go
entries, err := audit.Query(ctx, "tenant-1", time.Now().Add(-24*time.Hour))
for _, entry := range entries {
    fmt.Printf("%s: %s by %s (success=%v)\n",
        entry.Timestamp, entry.Action, entry.Actor, entry.Success)
}
```

## Key Expiry Alerts

### Monitoring

Run periodic checks (recommended: daily):

```go
expiring, err := cm.CheckExpiry(ctx, tenantID)
for _, cred := range expiring {
    daysUntilExpiry := time.Until(*cred.ExpiresAt).Hours() / 24
    if daysUntilExpiry <= 7 {
        // Send urgent alert
    }
}
```

### Alerting

Set up alerts for:
- **7 days before expiry**: Warning
- **3 days before expiry**: Urgent
- **1 day before expiry**: Critical

## Degraded Mode Behavior

When credentials are unavailable:

1. **Expired credentials**: Return error, block requests
2. **Revoked credentials**: Return error, block requests
3. **Missing credentials**: Fall back to default provider (if configured)
4. **Rotation in progress**: Use new credential, failover to old if needed

### Fallback Configuration

```go
router := NewPolicyRouter(registry,
    WithTenantStore(tenantStore),
)

// Tenant config with fallback
tenantStore.Set(&TenantConfig{
    TenantID:       "tenant-1",
    DefaultProvider: "openai",
    FallbackChain:  []ProviderID{"anthropic", "selfhosted"},
})
```

## Security Best Practices

1. **Never log API keys**: Credentials are masked in all logs
2. **Rotate regularly**: Set rotation reminders at 60-80% of key lifetime
3. **Use short-lived keys**: Prefer keys with 30-90 day expiry
4. **Audit regularly**: Review credential access logs weekly
5. **Revoke immediately**: On any suspicion of compromise

## Storage Backends

### In-Memory (Development)

```go
store := NewInMemoryCredentialStore()
```

### Production Backends

Implement `CredentialStore` interface for:
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- Kubernetes Secrets

Example interface:

```go
type CredentialStore interface {
    Get(ctx context.Context, tenantID string, providerID ProviderID) (*Credential, error)
    Set(ctx context.Context, cred *Credential) error
    Delete(ctx context.Context, tenantID string, providerID ProviderID) error
    List(ctx context.Context, tenantID string) ([]*Credential, error)
}
```

## Troubleshooting

### Credential Not Found

```
Error: credential not found for tenant X, provider Y
```

Solution: Create credential for tenant/provider combination

### Credential Expired

```
Error: credential expired
```

Solution: Rotate credential or extend expiry

### Credential Revoked

```
Error: credential revoked
```

Solution: Create new credential (cannot restore revoked credentials)

### Rotation Failed

```
Error: failed to store new credential
```

Solution: Check storage backend connectivity, retry rotation

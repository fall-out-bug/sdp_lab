# sdp-enterprise Helm Chart

This chart packages the adapter controller as a multi-tenant runtime with secure-by-default settings.

## Quick Validation

```bash
helm lint deploy/helm/sdp-enterprise
helm template sdp-enterprise deploy/helm/sdp-enterprise
helm upgrade --install sdp-enterprise deploy/helm/sdp-enterprise --namespace sdp-adapter --create-namespace --dry-run
```

## Upgrade / Rollback Flow (test cluster)

```bash
helm upgrade --install sdp-enterprise deploy/helm/sdp-enterprise --namespace sdp-adapter --create-namespace
helm upgrade sdp-enterprise deploy/helm/sdp-enterprise --namespace sdp-adapter --set tenants[0].image.tag=vNEXT
helm rollback sdp-enterprise 1 --namespace sdp-adapter
```

## Multi-tenant Values

Use `tenants[]` entries in `values.yaml` to define per-tenant namespace, replicas, image, NATS URL, and workspace volume strategy (`emptyDir` or PVC).

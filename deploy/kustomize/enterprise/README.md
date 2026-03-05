# Enterprise Kustomize Overlays

Environment overlays are provided for:

- `overlays/dev`
- `overlays/staging`
- `overlays/prod`

Validate rendered manifests:

```bash
kubectl kustomize deploy/kustomize/enterprise/overlays/dev >/tmp/sdp-enterprise-dev.yaml
kubectl kustomize deploy/kustomize/enterprise/overlays/staging >/tmp/sdp-enterprise-staging.yaml
kubectl kustomize deploy/kustomize/enterprise/overlays/prod >/tmp/sdp-enterprise-prod.yaml
```

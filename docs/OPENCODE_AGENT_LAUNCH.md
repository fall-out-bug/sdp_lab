# OpenCode Agent Launch (Private)

## Runtime contract

- runtime: `opencode`
- model: `glm-5`
- roles orchestrated: `swarm-worker` then `swarm-reviewer`

## Local launch

Single cycle:

```bash
go run ./cmd/opencode-agent
```

Continuous loop:

```bash
go run ./cmd/opencode-agent --loop --interval 30s
```

## k8s launch

Worker manifests include `opencode-agent` deployment:

```bash
./scripts/apply_worker_manifests.sh --host fall_out_bug@192.168.50.219 --port 2222
./scripts/check_remote_k8s.sh --host fall_out_bug@192.168.50.219 --port 2222
```

Publish runtime image before deployment:

```bash
./scripts/build_push_opencode_agent_image.sh
```

If local Docker engine is unavailable, use remote builder host:

```bash
./scripts/build_push_opencode_agent_image_remote.sh --host fall_out_bug@192.168.50.219 --port 2222
```

This remote script cross-compiles a linux/amd64 binary locally and builds a `scratch` image remotely.

## Notes

- Deployment expects image `ghcr.io/fall-out-bug/sdp-dev-opencode-agent:latest`.
- If image is unavailable, deployment will show `ErrImagePull` until image publication.

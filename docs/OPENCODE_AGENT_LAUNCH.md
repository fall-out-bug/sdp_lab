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

This remote script cross-compiles linux binaries locally into `.tmp/opencode-runtime`, syncs repository context, and builds a runtime image on the k8s host (`sdp-dev-opencode-agent:local`) without pushing to GHCR.
The runtime image includes `opencode-agent`, `swarm-worker`, `swarm-reviewer`, `autonomy-worker`, `beads-fsm`, `pr-gate`, `pr-publish`, and CLI dependencies (`bd`, `git`, `gh`).

## Notes

- Deployment uses local host image `sdp-dev-opencode-agent:local` with `imagePullPolicy: Never`.
- Deployment uses image `ENTRYPOINT` (`/usr/local/bin/opencode-agent --loop --interval 30s`) and does not override container command.

## Orchestrated k8s run

To orchestrate a specific task through in-cluster worker+reviewer and wait for terminal status:

```bash
./scripts/orchestrate_k8s_issue.sh --host fall_out_bug@192.168.50.219 --port 2222 --issue <issue-id>
```

Behavior:

- verifies deployment health
- runs `bd sync --import-only` inside the pod
- triggers an explicit single `opencode-agent` cycle
- polls task status (`closed` success, `blocked` failure)
- extracts PR URL from issue notes when present

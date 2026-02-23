# FR-016: Automated CI/CD Deployment Agent

Priority: P1
Effort: 4d
Dependencies: FR-015 (orchestrator creates AgentRuns), FR-002 (AgentRun CRD)

## Problem

Deployment is currently manual (scripts `deploy_swarm_platform.sh`, `build_push_opencode_agent_image.sh`). An agent is needed that:
- Watches for merged PRs
- Automatically builds images
- Deploys to target minikube host
- Verifies deployment
- Rolls back on problems

## Existing Code

| Component | Status | What it does |
|-----------|--------|-------------|
| `.github/workflows/ci.yml` | ✅ Active | Build, test, lint, image push (main only) |
| `scripts/deploy_swarm_platform.sh` | ✅ Active | Full platform deploy via SSH |
| `scripts/build_push_opencode_agent_image.sh` | ✅ Active | Build + push single image |
| `scripts/promote_opencode_agent_image_remote.sh` | ✅ Active | Promote image to remote |
| `scripts/bootstrap_remote_k8s.sh` | ✅ Active | Create namespaces |
| `scripts/apply_*_manifests.sh` | ✅ Active | Apply manifests per layer |

## Design

### Architecture

```
GitHub                      CI/CD Agent                    minikube host
──────                      ───────────                    ─────────────
PR merged to dev ──────►  Webhook/NATS ──► cicd-agent ──► SSH tunnel
                                            │               │
                          1. Detect change  │               │
                          2. Build images   │               ▼
                          3. Push to GHCR   │          kubectl apply
                          4. SSH deploy     │          kubectl rollout status
                          5. Health check   │          curl healthz
                          6. Verify         │          bd sync
                          7. Report         │
                                            ▼
                                      Beads: deployment issue closed
                                      NATS: sdp.deploy.{env}.{status}
```

### Agent Loop

```go
func (a *CICDAgent) Run(ctx context.Context) error {
    // Subscribe to deployment triggers
    a.bus.Subscribe("sdp.deploy.trigger.*", func(msg *nats.Msg) {
        trigger := parseTrigger(msg)
        a.deploy(ctx, trigger)
    })
    
    // Also watch for merged PRs via GitHub webhook → NATS
    a.bus.Subscribe("sdp.github.pr.merged", func(msg *nats.Msg) {
        pr := parsePR(msg)
        if pr.TargetBranch == "dev" || pr.TargetBranch == "main" {
            a.deploy(ctx, DeployTrigger{
                Project: pr.Repo,
                Ref:     pr.MergeCommit,
                Env:     envForBranch(pr.TargetBranch),
            })
        }
    })
}

func (a *CICDAgent) deploy(ctx context.Context, trigger DeployTrigger) error {
    // 1. Build images
    images := a.buildImages(trigger.Project, trigger.Ref)
    
    // 2. Push to registry
    a.pushImages(images)
    
    // 3. Update manifests with new image tags
    a.patchManifests(images)
    
    // 4. Deploy to target cluster
    a.applyManifests(trigger.Env)
    
    // 5. Wait for rollout
    a.waitRollout(trigger.Env, timeout)
    
    // 6. Health check
    if !a.healthCheck(trigger.Env) {
        a.rollback(trigger.Env)
        return ErrDeployFailed
    }
    
    // 7. Report
    a.report(trigger, "success")
}
```

### Target Environment

```yaml
environments:
  dev:
    host: minikube-host
    kubeconfig: /home/user/.kube/config
    ssh_key_secret: minikube-ssh-key
    namespaces:
      - sdp-control
      - sdp-workers
      - sdp-observability
    health_endpoints:
      - http://intake-gateway.sdp-control:8080/healthz
      - http://nats.sdp-control:8222/healthz
```

### Deployment Strategy

1. **Rolling update**: default for stateless services
2. **Recreate**: for stateful (NATS — already StatefulSet)
3. **Canary**: future (for risky deployments)
4. **Rollback**: automatic on failed health check

## Acceptance Criteria

- [ ] Agent runs in sdp-control namespace
- [ ] Triggers: NATS event (PR merged) or manual trigger
- [ ] Builds all service images (opencode-agent, orchestrator, intake-gateway, etc.)
- [ ] Pushes to GHCR with git SHA tag
- [ ] Deploys to remote minikube via SSH + kubectl
- [ ] Waits for rollout completion
- [ ] Health check post-deploy
- [ ] Automatic rollback on failure
- [ ] Beads issue for each deploy (type: chore)
- [ ] NATS events: deploy.started, deploy.succeeded, deploy.failed, deploy.rolled_back
- [ ] Evidence: deploy manifest, image SHAs, health check results
- [ ] ConfigMap: target env, SSH access, image registry

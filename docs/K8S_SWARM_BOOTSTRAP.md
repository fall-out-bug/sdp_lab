# k8s Swarm Bootstrap (Private)

Status: active draft
Goal: deploy swarm runtime to neighboring machine k8s cluster, managed from local control workspace over SSH.

## 1. Environment assumptions

- local control machine has SSH access to remote host
- remote host has working k8s context and cluster-admin for bootstrap
- secrets and policies are provisioned from private sources only

## 2. SSH control setup

Recommended local SSH config entry:

```sshconfig
Host sdp-k8s
  HostName <REMOTE_IP_OR_DNS>
  User <REMOTE_USER>
  IdentityFile ~/.ssh/<KEY_FILE>
  IdentitiesOnly yes
```

Validation:

```bash
ssh sdp-k8s "kubectl get nodes"
```

## 3. Namespace layout

- `sdp-control`
- `sdp-workers`
- `sdp-observability`
- `sdp-openclaw` (reserved for later)

Bootstrap:

```bash
ssh sdp-k8s "kubectl create ns sdp-control --dry-run=client -o yaml | kubectl apply -f -"
ssh sdp-k8s "kubectl create ns sdp-workers --dry-run=client -o yaml | kubectl apply -f -"
ssh sdp-k8s "kubectl create ns sdp-observability --dry-run=client -o yaml | kubectl apply -f -"
ssh sdp-k8s "kubectl create ns sdp-openclaw --dry-run=client -o yaml | kubectl apply -f -"
```

Scripted bootstrap:

```bash
./scripts/bootstrap_remote_k8s.sh --host fall_out_bug@192.168.50.219 --port 2222
```

## 4. Deployment order

1. control services
   - brain gateway
   - scheduler
   - policy service
2. workers
   - builder
   - verifier
   - reviewer
3. observability
   - metrics collector
   - evidence indexer
4. openclaw namespace (later stage)

## 5. Policy and model constraints

- hard model allowlist: `glm-5`, `glm-4.7`
- strict evidence gate required for PR publication
- protected branches denied for autonomous push

## 6. Operational checks

Apply baseline control-plane manifests:

```bash
./scripts/apply_control_manifests.sh --host fall_out_bug@192.168.50.219 --port 2222
```

Apply baseline worker manifests:

```bash
./scripts/apply_worker_manifests.sh --host fall_out_bug@192.168.50.219 --port 2222
```

Apply observability manifests:

```bash
./scripts/apply_observability_manifests.sh --host fall_out_bug@192.168.50.219 --port 2222
```

Health checks:

```bash
ssh sdp-k8s "kubectl -n sdp-control get deploy,pod"
ssh sdp-k8s "kubectl -n sdp-workers get deploy,pod"
ssh sdp-k8s "kubectl -n sdp-observability get deploy,pod"
```

Log checks:

```bash
ssh sdp-k8s "kubectl -n sdp-control logs deploy/brain-gateway --tail=200"
ssh sdp-k8s "kubectl -n sdp-workers logs deploy/verifier --tail=200"
```

Scripted health check:

```bash
./scripts/check_remote_k8s.sh --host fall_out_bug@192.168.50.219 --port 2222
```

Observability pipeline sanity check:

```bash
./scripts/sanity_check_observability_remote.sh --host fall_out_bug@192.168.50.219 --port 2222
```

## 7. Security notes

- do not embed secrets in manifests
- use remote secret manager or sealed secrets workflow
- keep private policy bundles out of OSS artifacts

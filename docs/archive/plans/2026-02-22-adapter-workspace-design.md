# Adapter Workspace Architecture

> **Status:** Design complete → FR-018 (sdp_dev-1es)
> **Date:** 2026-02-22
> **Goal:** Решить проблему workspace для adapter-controller: beads (.beads/), evidence (.sdp/), trace (.sdp/runs/) требуют persistent storage и доступ к проектным данным, но adapter использует emptyDir.

---

## Table of Contents

1. [Overview](#overview)
2. [Storage Topology](#1-storage-topology)
3. [Namespace Boundaries](#2-namespace-boundaries)
4. [Per-Project Workspace Routing](#3-per-project-workspace-routing)
5. [Graceful Degradation](#4-graceful-degradation)
6. [Evidence Persistence Strategy](#5-evidence-persistence-strategy)
7. [Beads CLI Dependency](#6-beads-cli-dependency)
8. [Kustomize Overlays](#7-kustomize-overlays)
9. [Implementation Plan](#implementation-plan)

---

## Overview

### Проблема

adapter-controller (sdp-adapter namespace) монтирует `/workspaces` как `emptyDir`. Из-за этого:
- **Beads:** `bd` вызовы не работают — нет `.beads/issues.jsonl`
- **Evidence:** `.sdp/evidence/` теряется при перезапуске pod
- **Traces:** `.sdp/runs/` теряются при перезапуске pod
- **Multi-project:** нет доступа к workspace'ам проектов (`/workspaces/<project_id>/`)

При этом swarm-orchestrator и feature-orchestrator в sdp-control используют общий PVC `swarm-workspaces`.

### Key Decisions

| Aspect | Decision |
|--------|----------|
| Storage topology | Переносим adapter в sdp-control, используем общий PVC |
| Namespace | adapter → sdp-control (control-plane компонент) |
| Per-project routing | WorkspaceResolver + per-call BeadsAdapter/EvidenceProjector |
| Graceful degradation | Startup health check, BeadsAdapter=nil когда workspace отсутствует |
| Evidence persistence | Краткосрочно: filesystem + PVC; долгосрочно: store interface → S3 |
| Beads CLI | Краткосрочно: добавить bd/beads-fsm в image; долгосрочно: Go API |
| Kustomize overlays | base + overlays (dev/staging/prod) для adapter |

---

## 1. Storage Topology

> **Experts:** Kelsey Hightower, Sam Newman, Martin Kleppmann

### Текущее состояние

| Компонент | Namespace | Volume | Mount |
|-----------|-----------|--------|-------|
| feature-orchestrator | sdp-control | swarm-workspaces PVC (RWO, 10Gi) | /workspaces |
| swarm-orchestrator | sdp-control | swarm-workspaces PVC | /workspaces |
| adapter-controller | sdp-adapter | **emptyDir** | /workspaces |

### Решение: adapter разделяет swarm-workspaces PVC

- Переносим adapter в sdp-control → один namespace, один PVC
- RWO: все pod'ы на одном node (minikube / single-node) — достаточно
- Multi-node: потребуется RWX (NFS / EFS) — отложено до масштабирования

### Риски

- **Write conflicts:** orchestrators и adapter пишут в `.beads/` и `.sdp/`. Решение: чёткое разделение ownership (orchestrators clone/update, adapter writes evidence/runs)
- **Multi-node:** RWO блокирует scheduling на другие ноды. Mitigation: node affinity или RWX позже

---

## 2. Namespace Boundaries

> **Experts:** Sam Newman, Kelsey Hightower, Troy Hunt

### Решение: adapter → sdp-control

adapter-controller — control-plane компонент (reconciles CRDs, drives beads/evidence). Логически принадлежит к sdp-control.

### RBAC

Добавить `adapter-controller-rbac.yaml` в sdp-workers (по аналогии с feature-orchestrator-rbac):

```yaml
# deploy/k8s/workers/adapter-controller-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: adapter-controller
  namespace: sdp-workers
rules:
  - apiGroups: ["sdp.dev"]
    resources: ["agentruns", "agentruns/status", "tasks", "tasks/status"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: adapter-controller
  namespace: sdp-workers
subjects:
  - kind: ServiceAccount
    name: adapter-controller
    namespace: sdp-control
roleRef:
  kind: Role
  name: adapter-controller
  apiGroup: rbac.authorization.k8s.io
```

### Изменения в deploy

| Файл | Действие |
|------|----------|
| `deploy/k8s/control/kustomization.yaml` | Добавить adapter-controller.yaml |
| `deploy/k8s/control/adapter-controller.yaml` | Новый: Deployment + ServiceAccount (namespace: sdp-control, volume: swarm-workspaces) |
| `deploy/k8s/workers/kustomization.yaml` | Добавить adapter-controller-rbac.yaml |
| `deploy/k8s/adapter/` | Архивировать или оставить для overlay dev (emptyDir) |

---

## 3. Per-Project Workspace Routing

> **Experts:** Sam Newman, Martin Kleppmann, Theo Browne

### Проблема

adapter использует один `WorkDir` для всех проектов. В multi-project mode каждый проект в `/workspaces/<project_id>/` со своими `.beads/` и `.sdp/`.

### Решение: WorkspaceResolver + per-call adapters

```go
// internal/adapter/workspace.go

const LabelProject = "sdp.project"

func ProjectIDFromLabels(labels map[string]string) string {
    if labels == nil { return "" }
    if v := labels[LabelProject]; v != "" { return v }
    return labels["project"]
}

type WorkspaceResolver func(projectID string) string

func NewWorkspaceResolver(baseDir string) WorkspaceResolver {
    return func(projectID string) string {
        if projectID == "" { projectID = "default" }
        return filepath.Join(baseDir, projectID)
    }
}
```

### Reconcile flow

```go
func (r *TaskReconciler) Reconcile(ctx, req) {
    // ...get task...
    projectID := ProjectIDFromLabels(task.Labels)
    workDir := r.WorkspaceResolver(projectID)
    beadsAdapter := beads.NewAdapter(workDir)
    projector := adapter.NewEvidenceProjector(workDir)
    // use per-call adapters for this reconcile
}
```

### Label propagation

Убедиться что `createTaskFromIntent` пробрасывает `sdp.project` из AgentRun в Task labels.

### TraceEmitter

Shared TraceEmitter не подходит для multi-project. Использовать `LoadTraceEventsFromRunFile(workDir, runID)` с per-project workDir. TraceEmitter — только fallback для single-project mode.

---

## 4. Graceful Degradation

> **Experts:** Martin Kleppmann, Sam Newman, Kelsey Hightower

### Решение: startup health check + conditional adapters

```go
// internal/adapter/workspace.go

type WorkspaceHealth struct {
    BeadsAvailable    bool
    BeadsFSMAvailable bool
    Reason            string
}

func CheckWorkspaceHealth(workDir string) WorkspaceHealth {
    h := WorkspaceHealth{}
    if _, err := exec.LookPath("bd"); err != nil {
        h.Reason = "bd not in PATH"; return h
    }
    if st, err := os.Stat(filepath.Join(workDir, ".beads")); err != nil || !st.IsDir() {
        h.Reason = ".beads/ absent"; return h
    }
    h.BeadsAvailable = true
    if _, err := exec.LookPath("beads-fsm"); err == nil {
        h.BeadsFSMAvailable = true
    }
    return h
}
```

### main.go

```go
health := adapter.CheckWorkspaceHealth(workDir)
var beadsAdapter *beads.Adapter
if health.BeadsAvailable {
    beadsAdapter = beads.NewAdapter(workDir)
} else {
    setupLog.Info("beads disabled", "reason", health.Reason)
}
```

### runBeadsFSM: no-op

```go
var beadsFSMAvailable = func() bool {
    _, err := exec.LookPath("beads-fsm")
    return err == nil
}()

func runBeadsFSM(workDir, issueID, target string) error {
    if !beadsFSMAvailable { return nil }
    // ...existing logic...
}
```

### Failure matrix

| Операция | `.beads/` нет | `bd` нет | Поведение |
|----------|---------------|----------|-----------|
| BeadsAdapter.Claim | Skip | Skip | BeadsAdapter=nil |
| BeadsAdapter.Close | Skip | Skip | nil |
| EvidenceProjector | Работает | N/A | os.MkdirAll |
| TraceEmitter | Работает | N/A | os.MkdirAll |
| runBeadsFSM | Skip | Skip | no-op |

---

## 5. Evidence Persistence Strategy

> **Experts:** Martin Kleppmann, Kelsey Hightower, Troy Hunt

### Retention requirements

| Artifact | Retention | Source |
|----------|-----------|--------|
| Evidence envelopes | 365–1825 дней | ARTIFACT_PROVENANCE_INTAKE |
| Run traces | Self-improvement reads каждые 6ч | self-improvement-contract.yaml |
| Provenance chain | hash → hash_prev → immutable | strict.go |

### Roadmap

| Phase | Backend | Когда |
|-------|---------|-------|
| Phase 0 (текущий) | Filesystem + emptyDir | Сейчас |
| **Phase 1** | Filesystem + PVC | Ближайшие 1-2 дня |
| Phase 2 | EvidenceStore/RunStore interface + filesystem impl | ~1 неделя |
| Phase 3 | S3/MinIO backend | Когда потребуется multi-cluster |

### Phase 2: Store interface

```go
type EvidenceStore interface {
    Write(projectID, issueID string, payload []byte) error
    Read(projectID, issueID string) ([]byte, error)
    List(projectID string) ([]string, error)
}

type RunStore interface {
    WriteRun(projectID, runID string, doc []byte) error
    ReadRun(projectID, runID string) ([]byte, error)
    ListRuns(projectID string) ([]string, error)
}
```

---

## 6. Beads CLI Dependency

> **Experts:** Kelsey Hightower, Sam Newman, Theo Browne

### Текущее состояние

- Dockerfile: `gcr.io/distroless/static-debian12:nonroot` — нет shell, нет `bd`, нет `beads-fsm`
- `beads.Adapter` → `exec.Command("bd", ...)`
- `runBeadsFSM` → `exec.Command("beads-fsm", ...)`

### Roadmap

| Phase | Подход | Когда |
|-------|--------|-------|
| **Phase 1** | Добавить `bd` и `beads-fsm` в image, сменить base на debian-slim | Ближайшие 1-2 дня |
| Phase 2 | Go API для JSONL read/write (inline beads) | ~2 недели |
| Phase 3 | NATS-based beads operations (worker performs beads) | Позже |

### Phase 1: Dockerfile

```dockerfile
FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/adapter-controller ./cmd/adapter-controller
RUN CGO_ENABLED=0 go build -o /out/beads-fsm ./cmd/beads-fsm

# Build bd from beads repo
RUN git clone https://github.com/steveyegge/beads /tmp/beads && \
    cd /tmp/beads && CGO_ENABLED=0 go build -o /out/bd ./cmd/bd

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates git && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/adapter-controller /usr/local/bin/
COPY --from=builder /out/bd /usr/local/bin/
COPY --from=builder /out/beads-fsm /usr/local/bin/
ENTRYPOINT ["adapter-controller"]
```

---

## 7. Kustomize Overlays

> **Experts:** Kelsey Hightower, Sam Newman, Martin Fowler

### Решение: base + overlays для adapter

```
deploy/k8s/adapter/
├── base/
│   ├── kustomization.yaml   # common: RBAC, namespace, CRDs
│   ├── deployment.yaml       # emptyDir (dev default)
│   ├── rbac.yaml
│   └── namespace.yaml
├── overlays/
│   ├── dev/                  # E2E: emptyDir, beads disabled
│   │   └── kustomization.yaml
│   ├── staging/              # PVC + bd in image
│   │   ├── kustomization.yaml
│   │   ├── volume-patch.yaml
│   │   └── adapter-workspaces-pvc.yaml
│   └── prod/                 # PVC + storageClass override
│       ├── kustomization.yaml
│       ├── volume-patch.yaml
│       └── adapter-workspaces-pvc.yaml
```

### Использование

```bash
# Dev/E2E
kubectl kustomize deploy/k8s/adapter/overlays/dev | kubectl apply -f -

# Staging/Prod (shared PVC)
kubectl kustomize deploy/k8s/adapter/overlays/staging | kubectl apply -f -
```

### Альтернатива при переносе в sdp-control

Если adapter переезжает в sdp-control, overlays не нужны — adapter просто добавляется в control kustomization и использует swarm-workspaces PVC. Overlay dev остаётся для E2E тестов с изолированным sdp-adapter namespace.

---

## Implementation Plan

### Phase 1: Graceful Degradation (0.5d) — P1

- [ ] `internal/adapter/workspace.go`: `CheckWorkspaceHealth`, `ProjectIDFromLabels`, `WorkspaceResolver`
- [ ] `cmd/adapter-controller/main.go`: startup health check, conditional BeadsAdapter
- [ ] `internal/adapter/task_reconciler.go`: `runBeadsFSM` no-op when unavailable
- [ ] Unit tests

### Phase 2: Adapter → sdp-control (1d) — P1

- [ ] `deploy/k8s/control/adapter-controller.yaml`: Deployment + ServiceAccount
- [ ] `deploy/k8s/control/kustomization.yaml`: добавить adapter-controller
- [ ] `deploy/k8s/workers/adapter-controller-rbac.yaml`: RBAC в sdp-workers
- [ ] Volume: swarm-workspaces PVC вместо emptyDir
- [ ] Обновить Dockerfile: добавить `bd` и `beads-fsm`
- [ ] Обновить `scripts/e2e_agentrun_minikube.sh`
- [ ] Архивировать `deploy/k8s/adapter/` → `deploy/k8s/adapter/overlays/dev/`

### Phase 3: Per-Project Routing (0.5d) — P2

- [ ] `TaskReconcilerOpts`: `WorkspaceResolver` вместо `WorkDir`
- [ ] `AgentRunReconcilerOpts`: `WorkspaceResolver`
- [ ] Per-call `BeadsAdapter`, `EvidenceProjector` в Reconcile
- [ ] Label propagation: `sdp.project` из AgentRun → Task
- [ ] Unit tests

### Phase 4: Evidence Store Interface (1d) — P2

- [ ] `internal/evidence/store.go`: `EvidenceStore`, `RunStore` interfaces
- [ ] Filesystem implementation
- [ ] Migrate writers/readers to use store
- [ ] Document PVC sizing

### Phase 5: Kustomize Overlays (0.5d) — P3

- [ ] Base + overlays structure для adapter
- [ ] CI validation для всех overlays

---

## Success Metrics

| Metric | Baseline | Target |
|--------|----------|--------|
| Beads operations work in adapter | No (emptyDir, no bd) | Yes (PVC, bd in image) |
| Evidence survives pod restart | No (emptyDir) | Yes (PVC) |
| Adapter starts without workspace | Crashes/noisy logs | Clean startup, beads disabled |
| Multi-project routing | Single workDir | Per-project workDir from labels |
| E2E on minikube | Works (no beads) | Works (with optional beads) |

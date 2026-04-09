# SDP Lab Control Workspace

Private build, planning, and orchestration workspace for SDP.
GitHub repo name: `sdp_lab`. Historical docs and bead IDs may still use `sdp_dev` as a legacy label for the same root repo.

## What This Repo Actually Does

`sdp_lab` is the private repo where we build and steer the SDP platform itself.

- platform code lives here: Go binaries, orchestration, evals, adapters, K8s manifests
- planning lives here: roadmap, workstreams, private design docs, execution runbooks
- public protocol artifacts live in `sdp/`: prompts, hooks, schemas, and OSS CLI work

If your goal is to **use SDP inside your own project**, this repo is not the primary onboarding surface. Start with [`sdp/docs/QUICKSTART.md`](sdp/docs/QUICKSTART.md).

## Rules

- This repo is the default place for strategic planning.
- Do not publish private architecture, enterprise scope, or commercial details into OSS repos.
- Export to OSS only through sanitized artifacts.

## Choose Your Path

| Goal | Start here |
|---|---|
| Understand what `sdp_lab` is and what lives here | [`docs/reference/project-map.md`](docs/reference/project-map.md) |
| Contribute to the platform or private lab runtime | [`AGENTS.md`](AGENTS.md), [`docs/MULTI-REPO-WORKFLOW.md`](docs/MULTI-REPO-WORKFLOW.md), [`docs/roadmap/ROADMAP.md`](docs/roadmap/ROADMAP.md) |
| Adopt SDP in a greenfield or brownfield project | [`sdp/docs/QUICKSTART.md`](sdp/docs/QUICKSTART.md), then [`sdp/README.md`](sdp/README.md) |

## IDE Support Today

- public onboarding flow is first-class for `Claude Code`, `Cursor`, and `OpenCode` / `Windsurf`
- `Codex` prompt compatibility exists in [`sdp/.codex/`](sdp/.codex/), but the public install flow is still manual rather than auto-detected
- if the question is "can I give SDP my keys and start working?", the honest answer lives in `sdp/`, not in the private-lab runbooks here

## Start Here

Use these as the canonical entrypoints:

1. `docs/reference/project-map.md` - what this repo is, where source of truth lives, and what to read first
2. `AGENTS.md` - operator rules, repo boundaries, branch policy, and delivery flow
3. `docs/MULTI-REPO-WORKFLOW.md` - `sdp_lab` vs `sdp` commit workflow
4. `docs/roadmap/ROADMAP.md` - current product direction and active feature train

This README is a broad inventory, not the canonical workflow doc.

## Main Components

- `cmd/`, `internal/` - platform binaries, orchestration, evals, kernel, adapters
- `deploy/` - deployable runtime and observability manifests
- `docs/roadmap/`, `docs/workstreams/`, `docs/plans/` - planning and execution surfaces
- `sdp/` - public submodule for prompts, hooks, schemas, and OSS CLI work

## Inventory

- `docs/architecture/REPO-BOUNDARY.md` - sdp vs sdp_lab boundary (binaries, publish mapping).
- `docs/MULTI-REPO-WORKFLOW.md` - parent repo vs submodule commit order, branch defaults, and recovery steps.
- `docs/PRIVATE_BLUEPRINT.md` - full private architecture and roadmap.
- `docs/OSS_EXPORT_TEMPLATE.md` - sanitized structure for public RFCs.
- `docs/REDACTION_RULES.md` - what must never leak to OSS.
- `docs/OPENCODE_BRAIN_INTEGRATION_PLAN.md` - Stage A execution plan.
- `docs/K8S_SWARM_BOOTSTRAP.md` - remote cluster bootstrap over SSH.
- `docs/OBSERVABILITY_STACK_DEPLOY_RUNBOOK.md` - deploy/sanity workflow for Prometheus/Loki/Tempo/Grafana stack.
- `docs/OPENCLAW_ADAPTER_PLAN.md` - Stage B parity plan.
- `docs/ADR-0001-go-first-stack.md` - stack decision: Go-first, Python research lane.
- `docs/PR_GATE_RUNBOOK.md` - strict evidence gate workflow.
- `docs/REPO_BOUNDARY_MAP.md` - private/OSS capability boundary map.
- `docs/REDACTION_CHECKLIST.md` - pre-export redaction steps.
- `docs/CONTRACT_PARITY_REPORT.md` - OpenCode/OpenClaw contract parity baseline.
- `docs/GIT_REMOTE_BOOTSTRAP.md` - initialize remote default branch for PR publishing.
- `docs/REAL_FEATURE_TO_PR_RUNBOOK.md` - live operator flow from task claim to PR.
- `docs/OPENCODE_AGENT_LAUNCH.md` - launch procedure for opencode agent runtime.
- `docs/FEATURE_SHORTCUT_RUNBOOK.md` - one-command path from feature request to PR URL.
- `docs/MULTI_ROLE_OPERATOR_ORCHESTRATION.md` - multi-agent role orchestration and communication contract.
- `docs/KUBEOPENCODE_MULTI_ROLE_PROBE_RUNBOOK.md` - operator multi-role probe commands and known blockers.
- `docs/PARALLEL_LOCK_DOMAIN_INTAKE.md` - lock domains and hazard catalog for parallel execution controls.
- `docs/PARALLEL_SCHEDULER_POLICY.md` - lock hierarchy, merge queue semantics, incident safeguards, and tuning guidance.
- `docs/EVALUATOR_INTAKE_BASELINE.md` - happy-path operability gate and baseline evaluator scope.
- `docs/EVALUATOR_SWARM_DEEP_THINKING_PLAN.md` - deep-thinking evaluator cycle, persona roles, and collaboration protocol.
- `docs/EVALUATOR_SWARM_RUNTIME_ORCHESTRATION.md` - persona execution packet contract and score/report assembly flow.
- `docs/EVALUATOR_PERIODIC_COMPONENT_AUDIT_PROTOCOL.md` - repeatable periodic component audit protocol with checkpoints and escalation flow.
- `docs/EVALUATOR_OUTCOME_SCORING_RUBRIC.md` - weighted rubric and normalization contract for ranking improvement opportunities.
- `docs/EVALUATOR_TRIAL_RUN_CALIBRATION.md` - deterministic trial-run methodology, quality thresholds, and calibration evidence format.
- `docs/EVALUATOR_PR_LOOP_BACKLOG_INJECTION.md` - continuous-improvement PR-loop contract and deterministic backlog-injection guardrails.
- `docs/AGENT_ARTIFACT_COMMUNICATION_PROTOCOL.md` - semantic success gates and artifact communication protocol.
- `docs/OBSERVABILITY_METRICS_TRACE_SCHEMA_INTAKE.md` - unified metrics+trace schema for system/protocol/model tags.
- `specs/autonomy-runtime-contract.yaml` - runtime contract baseline.
- `specs/brain-decision-api.yaml` - brain decision request/response contract.
- `specs/strict-evidence-template.json` - mandatory PR evidence structure.
- `cmd/sdp/` - main SDP CLI: `discover`, `dispatch`, `board`, `tower`, `doctor`, `pipeline`.
- `cmd/sdp-orchestrate/` - oneshot outer loop: `--advance`, `--status`, `--next-action`.
- `cmd/sdp-evidence/` - standalone evidence CLI: `validate` and `inspect` subcommands. Zero K8s dependency.
- `cmd/sdp-dispatch/` - routing and profiling for harness dispatch.
- `cmd/sdp-ci-loop/` - CI feedback loop with deterministic autofix.
- `cmd/sdp-eval/` - evaluation framework runner.
- `cmd/sdp-guard/` - permission scope gate for agent invocations.
- `cmd/sdp-omc-guard/` - OMO client guard for tool policy enforcement.
- `cmd/sdp-beads-bridge/` - Beads issue tracker bridge.
- `cmd/sdp-ready/` - find ready work from Beads queue with SDP mapping.
- `cmd/sdp-protocol-check/` - validate SDP protocol hygiene across files.
- `cmd/sdp-doc-sync/` - documentation consistency and changelog automation.
- `cmd/sdp-a2a/` - agent-to-agent communication server.
- `cmd/sdp-control/` - control plane CLI.
- `cmd/sdp-ws-verdict-validate/` - workstream verdict validation.
- `cmd/sdp-orchestrate-daemon/` - long-running orchestration daemon.
- `cmd/sdp-gh-findings-sync/` - sync GitHub findings into local Beads queue.
- `cmd/sdp-up/` - bootstrap and deploy SDP components.
- `scripts/bootstrap_remote_k8s.sh` - creates required namespaces on remote cluster via SSH.
- `scripts/check_remote_k8s.sh` - runs namespace health checks on remote cluster via SSH.
- `scripts/apply_control_manifests.sh` - applies baseline control-plane manifests to remote cluster.
- `scripts/apply_worker_manifests.sh` - applies baseline worker manifests to remote cluster.
- `scripts/apply_observability_manifests.sh` - applies observability stack manifests to remote cluster.
- `scripts/sanity_check_observability_remote.sh` - validates observability telemetry pipeline on remote cluster.
- `scripts/build_push_opencode_agent_image.sh` - builds and pushes opencode-agent image to GHCR.
- `scripts/build_push_opencode_agent_image_remote.sh` - builds opencode-agent image on remote host for local k8s runtime.
- `scripts/orchestrate_k8s_issue.sh` - triggers an in-cluster agent cycle and waits for issue close/blocked with PR extraction.
- `scripts/feature_to_pr.sh` - creates a task and orchestrates k8s worker+reviewer to return PR URL.
- `scripts/install_kubeopencode_remote.sh` - installs/upgrades kubeopencode operator on remote cluster.
- `scripts/remote_minikube_tunnel.sh` - SSH tunnel to remote minikube; use local kubectl with `KUBECONFIG=.kube/remote-minikube.yaml`.
- `scripts/run_kubeopencode_multi_role_probe.sh` - runs analyst/coder/reviewer operator task probe and prints summary.
- `deploy/images/opencode-agent/Dockerfile.runtime` - runtime image with agent binaries plus `bd`/`git`/`gh` for in-pod execution.
- `deploy/k8s/observability/` - deployable observability stack and telemetry ingestion pipeline manifests.

## sdp-evidence CLI

Standalone binary for validating and inspecting evidence envelopes. No K8s dependency.

### Install

**From GitHub Releases** (after tagging e.g. `v0.1.0`):

```bash
# Linux amd64
curl -sSL https://github.com/OWNER/REPO/releases/download/v0.1.0/sdp-evidence_0.1.0_linux_amd64.tar.gz | tar xz -C /usr/local/bin

# macOS (darwin/arm64)
curl -sSL https://github.com/OWNER/REPO/releases/download/v0.1.0/sdp-evidence_0.1.0_darwin_arm64.tar.gz | tar xz -C /usr/local/bin
```

**From source:**

```bash
go install ./cmd/sdp-evidence@latest
```

### Usage

```bash
sdp-evidence validate --evidence .sdp/evidence/run-123.json
sdp-evidence inspect --evidence .sdp/evidence/run-123.json
```

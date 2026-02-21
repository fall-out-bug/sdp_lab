# SDP Dev Control Workspace

Private planning and orchestration workspace for SDP evolution.

## Rules

- This repo is the default place for strategic planning.
- Do not publish private architecture, enterprise scope, or commercial details into OSS repos.
- Export to OSS only through sanitized artifacts.

## Folders

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
- `cmd/autonomy-worker/` - picks next autonomy task from Beads and prepares execution packet. Modular: main_types, main_picker, main_labels, main_io, main_evidence, main_observability.
- `cmd/brain-gateway/` - evaluates policy/risk/model/branch decision.
- `cmd/beads-fsm/` - validates/applies guarded state transitions.
- `cmd/pr-gate/` - blocks PR progression when strict evidence is incomplete.
- `cmd/pr-publish/` - creates PR via `gh` and writes `trace.pr_url` into evidence.
- `cmd/swarm-worker/` - worker role that claims and implements eligible coding tasks. Modular: main_flow, main_handlers, main_patches, main_ensure_*, main_verify*, main_observability, main_runner, main_util. See docs/SWARM_WORKER_RUNBOOK.md.
- `cmd/swarm-reviewer/` - reviewer role that validates review flow and finalizes tasks.
- `cmd/opencode-agent/` - orchestrates worker+reviewer cycle using OpenCode with model routing (`swarm-worker`: `glm-4.7`, `swarm-reviewer`: `glm-5`).
- `cmd/redaction-check/` - scans candidate OSS exports for forbidden private terms.
- `cmd/runtime-parity-check/` - compares runtime capability sets for contract parity.
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
- `scripts/run_kubeopencode_multi_role_probe.sh` - runs analyst/coder/reviewer operator task probe and prints summary.
- `deploy/images/opencode-agent/Dockerfile.runtime` - runtime image with agent binaries plus `bd`/`git`/`gh` for in-pod execution.
- `deploy/k8s/observability/` - deployable observability stack and telemetry ingestion pipeline manifests.

- `cmd/flow-inspect/` - inspects protocol flow state from run packets.

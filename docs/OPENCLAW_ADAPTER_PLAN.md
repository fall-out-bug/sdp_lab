# OpenClaw Adapter Plan (Stage B)

Status: queued after Stage C
Goal: achieve runtime parity with OpenCode through a shared autonomy contract.

## 1. Adapter purpose

OpenClaw adapter must execute the same protocol semantics as OpenCode:

- same Beads state transitions
- same strict evidence requirements
- same model policy restriction
- same escalation behavior

## 2. Shared contract

Target contract name: `AutonomousRuntimeModule`

Required operations:

- `claimTask(issueID)`
- `loadTask(issueID)`
- `createBranch(issueID, slug)`
- `executeTask(plan)`
- `runVerification(taskCtx)`
- `buildEvidence(taskCtx)`
- `publishPR(taskCtx)`
- `updateTaskState(issueID, state, payload)`
- `escalate(issueID, reason)`

## 3. Compatibility requirements

- branch naming parity with OpenCode
- identical evidence schema keys
- identical retry/fallback class behavior
- identical escalation payload format

## 4. Validation plan

Test matrix:

1. user-feature scenario on OpenCode and OpenClaw produce equivalent outputs
2. agent-initiated scenario on both runtimes produce equivalent outputs
3. model policy violation on both runtimes escalates identically
4. missing evidence section blocks PR on both runtimes

## 5. k8s deployment tie-in

- OpenClaw runtime deploys in `sdp-openclaw` namespace
- adapter sidecar connects to `sdp-control` brain gateway
- all events are exported through shared trace pipeline

## 6. Exit criteria

- one green parity report for each scenario
- no contract drift between runtimes
- no policy bypass in OpenClaw path

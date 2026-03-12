# Reality C4 Components

- Generated At: `2026-03-12T13:21:35Z`
- Container Scope: `container:sdp_dev`

## Components

### `sdp/cmd`

- ID: `component:repo:sdp:cmd`
- Technology: `Go`
- Summary: cmd contains 13 source files.
- Responsibilities: Exposes executable entrypoints
- Paths: `cmd/sdp-ci-loop/main.go`, `cmd/sdp-ci-loop/main_test.go`, `cmd/sdp-eval/main.go`, `cmd/sdp-eval/main_test.go`, `cmd/sdp-evidence/main.go`, `cmd/sdp-evidence/main_test.go`

### `sdp/internal`

- ID: `component:repo:sdp:internal`
- Technology: `Go`
- Summary: internal contains 68 source files.
- Responsibilities: Implements core runtime behavior; Interfaces: cli
- Paths: `internal/ciloop/autofixer.go`, `internal/ciloop/autofixer_test.go`, `internal/ciloop/checkpoint.go`, `internal/ciloop/checkpoint_test.go`, `internal/ciloop/classifier.go`, `internal/ciloop/classifier_test.go`

### `sdp/scripts`

- ID: `component:repo:sdp:scripts`
- Technology: `Shell`
- Summary: scripts contains 15 source files.
- Responsibilities: Automates repeatable maintenance and quality tasks
- Paths: `scripts/beads_export.sh`, `scripts/beads_import_only.sh`, `scripts/check-workstream-beads-consistency.sh`, `scripts/check_complexity.sh`, `scripts/commit-with-coauthor.sh`, `scripts/install-project.sh`

### `sdp/sdp-plugin`

- ID: `component:repo:sdp:sdp-plugin`
- Technology: `Go`
- Summary: sdp-plugin contains 593 source files.
- Responsibilities: Carries protocol runtime and CLI integration logic; Interfaces: cli, handler
- Paths: `sdp-plugin/cmd/sdp/acceptance.go`, `sdp-plugin/cmd/sdp/acceptance_test.go`, `sdp-plugin/cmd/sdp/apply.go`, `sdp-plugin/cmd/sdp/beads.go`, `sdp-plugin/cmd/sdp/beads_test.go`, `sdp-plugin/cmd/sdp/build.go`

### `sdp/src`

- ID: `component:repo:sdp:src`
- Technology: `Markdown and JSON Schema`
- Summary: src contains 87 source files.
- Responsibilities: Feeds downstream service repos with protocol assets; src contains 87 source files.
- Paths: `src/sdp/agents/code_analyzer.go`, `src/sdp/agents/code_analyzer_backend.go`, `src/sdp/agents/code_analyzer_contract.go`, `src/sdp/agents/code_analyzer_frontend.go`, `src/sdp/agents/code_analyzer_test.go`, `src/sdp/agents/contract_generator.go`

### `sdp_dev/cmd`

- ID: `component:repo:sdp_dev:cmd`
- Technology: `Go`
- Summary: cmd contains 35 source files.
- Responsibilities: Exposes executable entrypoints
- Paths: `cmd/sdp-beads-bridge/main.go`, `cmd/sdp-beads-bridge/query.go`, `cmd/sdp-ci-loop/main.go`, `cmd/sdp-ci-loop/main_test.go`, `cmd/sdp-doc-sync/main.go`, `cmd/sdp-eval/main.go`

### `sdp_dev/docs`

- ID: `component:repo:sdp_dev:docs`
- Technology: `Shell`
- Summary: docs contains 2 source files.
- Responsibilities: Captures operator-facing intent and guidance
- Paths: `docs/integrations/audit-log.sh`, `docs/integrations/telegram.sh`

### `sdp_dev/internal`

- ID: `component:repo:sdp_dev:internal`
- Technology: `Go`
- Summary: internal contains 168 source files.
- Responsibilities: Implements core runtime behavior; Interfaces: cli
- Paths: `internal/adapters/sdk/contracts.go`, `internal/adapters/sdk/examples/main.go`, `internal/adapters/sdk/interfaces.go`, `internal/adapters/sdk/validation.go`, `internal/adapters/sdk/validation_test.go`, `internal/authz/tenant_scope.go`

### `sdp_dev/scripts`

- ID: `component:repo:sdp_dev:scripts`
- Technology: `Shell`
- Summary: scripts contains 42 source files.
- Responsibilities: Automates repeatable maintenance and quality tasks
- Paths: `scripts/apply_control_manifests.sh`, `scripts/apply_observability_manifests.sh`, `scripts/apply_worker_manifests.sh`, `scripts/autonomy_slo_report.sh`, `scripts/beads_export.sh`, `scripts/beads_import_only.sh`

### `sdp_dev/tests`

- ID: `component:repo:sdp_dev:tests`
- Technology: `Go workspace`
- Summary: tests contains 1 source files.
- Responsibilities: Verifies behavior at repository level
- Paths: `tests/contracts/compatibility_test.go`

## Relationships

- `component:repo:sdp:cmd` -> `component:repo:sdp:internal`: invokes runtime packages
- `component:repo:sdp:cmd` -> `component:repo:sdp:sdp-plugin`: surfaces plugin runtime
- `component:repo:sdp:scripts` -> `component:repo:sdp:cmd`: automates command workflows
- `component:repo:sdp:src` -> `component:repo:sdp:sdp-plugin`: provides assets consumed by plugin runtime
- `component:repo:sdp_dev:cmd` -> `component:repo:sdp_dev:internal`: invokes runtime packages
- `component:repo:sdp_dev:docs` -> `component:repo:sdp_dev:internal`: documents runtime behavior
- `component:repo:sdp_dev:internal` -> `component:repo:sdp:internal`: consumes protocol contracts and skills
- `component:repo:sdp_dev:scripts` -> `component:repo:sdp_dev:cmd`: automates command workflows
- `component:repo:sdp_dev:tests` -> `component:repo:sdp_dev:internal`: verifies runtime paths

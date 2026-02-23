# Feature: CRD Type Definitions + Code Generation (FR-003)

Priority: P0
Effort: 2 days
Blocks: FR-001, FR-002

## Problem

adapter-controller uses hand-rolled Go structs (CRDPhase, TaskIntent) instead of real CRD types. No informer, no typed clientset. Without code-gen it is impossible to write the controller.

## Scope

1. Create `api/v1alpha1/types.go`:
   - AgentRun CRD (spec + status)
   - Reuse kubeopencode Task/Agent types via shim interface
2. Add kubebuilder markers
3. Run controller-gen for DeepCopy
4. Add controller-runtime to go.mod
5. Typed clientset and informers

## Acceptance Criteria

- `go build ./api/...` passes
- `make generate` creates zz_generated.deepcopy.go
- Types are compatible with internal/adapter/ (LifecycleReconciler accepts CRDPhase)

## Dependencies

- sigs.k8s.io/controller-runtime
- sigs.k8s.io/controller-tools (controller-gen)

## Risks

- If UP-001 changes the CRD schema, the shim will require updates

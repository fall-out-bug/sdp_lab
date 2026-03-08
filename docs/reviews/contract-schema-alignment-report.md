# Contract/State/Schema Alignment Report

**Date:** 2026-03-08  
**Scope:** `Recommendation`, `StatusView`, parser frontmatter fields, and related tests  
**Context:** Added `action_id`, context/evidence/policy fields, status view with workstream arrays and environment flags.

---

## 1. Recommendation Contract Alignment

### 1.1 Go Struct vs JSON Schema

| Field | contract.go | instructions.schema.json | status-view.schema.json (instruction_payload) | Notes |
|-------|-------------|--------------------------|-----------------------------------------------|-------|
| action_id | `omitempty` | required, minLength:1 | required, minLength:1 | **Mismatch:** Go allows empty; schema requires non-empty |
| command | required | required | required | OK |
| reason | required | required | required | OK |
| confidence | required | required | required | OK |
| category | required | required | required | OK |
| version | required | required | required | OK |
| alternatives | omitempty | optional | optional | OK |
| required_context | omitempty | optional | optional | OK |
| optional_context | omitempty | optional | optional | OK |
| policy_expectations | omitempty | optional | optional | OK |
| evidence_expectations | omitempty | optional | optional | OK |
| metadata | omitempty | optional | optional | OK |

**Mitigation:** `enrich()` populates `action_id` before output. `ToJSON()` and `Resolver.Recommend()` both call `enrich()`. Direct `json.Marshal(rec)` without `enrich()` would produce schema-invalid output.

**Files:**
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/sdp-plugin/internal/nextstep/contract.go` (lines 43, 108-124)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/schema/contracts/instructions.schema.json` (lines 8, 10)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/schema/contracts/status-view.schema.json` (lines 95-97)

---

## 2. StatusView Contract Alignment

### 2.1 Go Struct vs status-view.schema.json

| Field | status_view.go | status-view.schema.json | Notes |
|-------|----------------|-------------------------|-------|
| next_action | `string` (can be "") | required, minLength:1 | **Mismatch:** When `NextStep` is nil, `NextAction` is ""; schema requires non-empty |
| next_step | `*Recommendation`, omitempty | required | **Mismatch:** When nil, omitted from JSON; schema requires presence |

**Root cause:** Schema assumes `next_step` and `next_action` are always present and non-empty. Go allows `BuildStatusView(..., nil)` which yields `NextStep == nil`, `NextAction == ""`.

**Current behavior:** `Resolver.Recommend()` always returns non-nil, so in production `NextStep` is never nil. Schema validation would fail only if `BuildStatusView` is called directly with `nil` or if resolver behavior changes.

**Files:**
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/sdp-plugin/internal/nextstep/status_view.go` (lines 18-20, 94-96)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/schema/contracts/status-view.schema.json` (lines 8, 68-70)

### 2.2 Environment Flags

`EnvironmentView` and `WorkstreamView` align with schema. No mismatches found.

---

## 3. Parser Frontmatter Alignment

### 3.1 Parser Schema vs Protocol Validator vs Shell Script

| Field | parser/schema.go (frontmatter) | protocol_validate.go | validate-workstream.sh | AGENTS.md / sdp-protocol-check |
|-------|-------------------------------|----------------------|------------------------|--------------------------------|
| ws_id | required | required | required | required |
| feature / feature_id | feature OR feature_id | feature_id | feature (not feature_id) | feature_id |
| status | required | required | required | required |
| priority | int | required | — | required |
| size | required | required | required | required |
| depends_on | optional | required | — | required |
| project_id | optional | — | — | — |
| goal | (from body) | — | required | — |
| AC | (from body) | — | required | — |
| context | — | — | required | — |
| steps | — | — | required | — |

**Mismatches:**
1. **Shell script vs protocol:** `validate-workstream.sh` requires `feature`, `goal`, `AC`, `context`, `steps`; protocol requires `feature_id`, `priority`, `depends_on`. Shell does not check `priority` or `depends_on`.
2. **Shell script vs protocol:** Shell requires `context`, `steps`; these are not in protocol frontmatter.
3. **Status values:** Shell allows `backlog`, `active`, `completed`, `blocked`. Protocol/collector use `backlog`, `ready`, `in_progress`, `blocked`, `completed`, `failed`. `active` vs `in_progress`/`ready` mismatch.
4. **Size values:** Shell expects `SMALL`, `MEDIUM`, `LARGE`. Backlog files use `M`, `S`, `L` (e.g. 00-069-01: `size: M`). Shell would reject `M`.

**Files:**
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/sdp-plugin/internal/parser/schema.go` (lines 52-63)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/workstream/protocol_validate.go` (lines 159-176)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/hooks/validate-workstream.sh` (lines 34-38, 88-124)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/AGENTS.md` (line 209)

### 3.2 Priority Type Mismatch (Critical)

| Source | Type | Example |
|--------|------|---------|
| parser frontmatter | `int` | `priority: 0` |
| Backlog files | string (P1, P2) | `priority: P1` (00-068-03, 00-069-04, etc.) |

**Risk:** YAML unmarshaling `"P1"` into `int` fails. `ParseWorkstream` returns error; collector skips the file (`if err != nil { continue }`). Workstreams with `priority: P1` are silently dropped from status.

**Files:**
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/sdp-plugin/internal/parser/schema.go` (line 58)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/sdp/sdp-plugin/internal/nextstep/collector.go` (lines 60-62)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/docs/workstreams/backlog/00-068-03.md` (line 5)

---

## 4. WorkstreamStatus vs Schema

`WorkstreamStatus` in `types.go` aligns with `workstream_status` in status-view.schema.json. Fields: `id`, `status`, `priority`, `blocked_by`, `feature`, `size`, `last_error`. No mismatches.

---

## 5. Contract-Test Risks

### 5.1 Tests That Rely on enrich()

| Test | File | Risk |
|------|------|------|
| TestNextStepRecommendationSchema | contract_test.go | Calls `rec.enrich()` before `json.Marshal`; OK |
| TestConsumerSurfaceParsing | contract_test.go | Calls `rec.enrich()`; checks `action_id` in parsed output; OK |
| TestBuildStatusView | status_view_test.go | Uses enriched rec; OK |
| TestStatusAndInstructionSchemasValidateContracts | status_view_test.go | Uses enriched rec; validates against schema; OK |
| TestStatusCmdJSONMode | status_test.go | Asserts `view.NextStep.ActionID != ""`; OK |
| TestNextCommandJSON | next_test.go | Asserts `rec.ActionID != ""`; OK |
| TestNextCommandWithWorkstreams | next_test.go | Asserts `rec.ActionID != ""`; OK |

**Risk:** Tests that construct `Recommendation` without calling `enrich()` and then marshal would fail schema validation. No such tests found.

### 5.2 Tests That Assume next_step Always Present

| Test | File | Risk |
|------|------|------|
| TestStatusAndInstructionSchemasValidateContracts | status_view_test.go | Always passes non-nil rec; does not cover nil NextStep |
| TestStatusCmdJSONMode | status_test.go | Asserts `view.NextStep != nil`; fixture always produces recommendation |
| TestPrintStatusJSON | status_test.go | Constructs view with non-nil NextStep |

**Risk:** No test validates StatusView when `NextStep` is nil. Schema would fail; no regression test for that path.

### 5.3 Parser Tests and Priority

| Test | File | Fixture priority | Risk |
|------|------|------------------|------|
| TestParseValidWorkstream | workstream_test.go | (none) | No priority in fixture |
| TestParseWorkstreamWithFeatureID | workstream_test.go | (none) | No priority |
| TestParseWorkstreamWithPriorityAndDeps | workstream_test.go | `priority: 2` (int) | OK |

**Risk:** No test uses `priority: P1`. P1 parsing failure is untested.

---

## 6. Registry / Documentation References

- `docs/reference/integration-contracts.md`: Documents `next_action` and `next_step` consistency.
- `docs/reference/schema-registry.md`: References `TestSchemaRegistryLoads` in parser.
- `AGENTS.md`: Lists frontmatter required fields; matches protocol_validate.go.

---

## 7. Summary of Mismatches

| Severity | Issue | Location |
|----------|-------|----------|
| **High** | `priority: P1` (string) cannot parse into `int`; workstreams silently skipped | parser schema, collector |
| **Medium** | StatusView schema requires `next_step` and `next_action`; Go allows nil/empty | status_view.go, status-view.schema.json |
| **Medium** | Recommendation `action_id` has `omitempty` but schema requires it | contract.go, instructions.schema.json |
| **Low** | validate-workstream.sh frontmatter set diverges from protocol | validate-workstream.sh |
| **Low** | Shell status enum (`active`) vs protocol (`ready`, `in_progress`) | validate-workstream.sh |
| **Low** | Shell size enum (`SMALL`, `MEDIUM`, `LARGE`) vs backlog (`M`, `S`) | validate-workstream.sh |

---

## 8. Recommended Next Steps

1. **Priority parsing:** Support `priority: P1` (e.g. map P0→0, P1→1, P2→2) or document that numeric priority is required and migrate backlog.
2. **StatusView schema:** Either relax schema (make `next_step`/`next_action` optional) or guarantee non-nil in `BuildStatusView` and add a test for nil case.
3. **validate-workstream.sh:** Align with protocol (feature_id, priority, depends_on) or deprecate in favor of sdp-protocol-check.
4. **Add regression test:** `TestStatusViewWithNilNextStep` to document and validate behavior when no recommendation exists.

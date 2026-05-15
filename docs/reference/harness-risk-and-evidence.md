# Harness Risk And Evidence

Status: reference

This vocabulary keeps SDP skill and harness claims honest. It is intentionally
small: use it in skills, review reports, adapter checks, and evidence summaries
without turning every task into a policy project.

## Tool Risk Classes

| Class | Meaning | Default policy |
|---|---|---|
| `perception` | Read-only inspection: files, logs, docs, links, local state. | Allowed for most roles. |
| `analysis` | Local computation or synthesis without writes or external side effects. | Allowed with recorded evidence. |
| `local_write` | Edits, generated artifacts, local database or checkpoint changes. | Implementer/workflow scope only. |
| `external_write` | Push, publish, create or update a remote system, send messages. | Explicit workflow gate required. |
| `irreversible` | Merge, deploy, delete, rotate credentials, spend money. | Explicit human or workflow authorization required. |

Prompt text may describe a boundary, but it is not the boundary. If the harness
cannot enforce a risk-class gate, record the claim as `not_assessed_runtime` or
`manual_gate_only`.

## Evidence States

| State | Meaning |
|---|---|
| `passed` | Evidence completed and supports the claim. |
| `failed` | Evidence completed and contradicts the claim. |
| `not_assessed` | The plane was not run. |
| `failed_provider` | Provider returned an explicit error. |
| `timeout` | Run exceeded the bounded window. |
| `empty_output` | Run completed with no useful content. |
| `off_task` | Output did not address the requested plane. |
| `unavailable_cli` | Required local tool was missing or could not run. |
| `unverified_benchmark` | Vendor or third-party claim was not validated on SDP tasks. |
| `not_assessed_runtime` | Static files exist, but runtime behavior was not proven. |
| `manual_gate_only` | The workflow used an explicit human/workflow gate because runtime enforcement is unavailable. |

Missing evidence is not a pass. Use the degraded state that preserves what
actually happened.

## Assignment Rule

Deterministic tool output wins over model prose. If a model claims a check
passed but the tool output is missing, classify the check as `not_assessed`.
If states conflict, report the more conservative degraded state until a human or
orchestrator inspects the evidence.

## Common Examples

- A review provider returns no findings because the process timed out:
  `timeout`, not `passed`.
- A harness adapter file exists but no dispatch run proves it loads:
  `not_assessed_runtime`.
- A skill says "do not push", but the harness cannot block pushing:
  `manual_gate_only` for that action class unless another runtime gate exists.
- A model vendor page claims strong coding benchmarks:
  `unverified_benchmark` until reproduced on SDP tasks.

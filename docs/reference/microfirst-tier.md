# MicroFirst Inference Tier

## What it is

Pre-LLM gate that short-circuits expensive inference for obvious cases.
Uses `decompose.WithEscalation(micro, llm, cfg)` Stage-composer.

The MicroFirst tier runs lightweight embedding-based kNN classifiers first.
If confidence exceeds the configured threshold, the result is returned immediately
without invoking the LLM. Only ambiguous cases escalate to the full LLM stage.

## How to add a new micro Stage

1. Create package `internal/inference/microfirst/<name>/`
2. Implement `Stage[In, Out]` where `Out` implements `decompose.Confider`
3. Return `Status=OK` with high confidence for confident cases; `StatusUnsure` otherwise
4. Wrap with `decompose.WithEscalation(yourMicro, yourLLM, EscalationConfig{ConfidenceThreshold: 0.85})`

### Example structure

```go
package mymicro

type MyInput struct { Title string }
type MyResult struct {
    Label      string
    confidence float64
    status     decompose.Status
}
func (r MyResult) Confidence() float64        { return r.confidence }
func (r MyResult) ConfStatus() decompose.Status { return r.status }

type MyMicro struct { /* embedder + knn index */ }
func (m *MyMicro) Name() string { return "my-micro" }
func (m *MyMicro) Run(ctx context.Context, in MyInput) (MyResult, decompose.StageTrace, error) { ... }
```

## CLI: bd suggest

```bash
sdp-bd-suggest --title="fix nil pointer in dispatcher" \
               --description="crash on startup" \
               [--format=json|human] \
               [--ollama-url=http://localhost:11434] \
               [--corpus-path=.beads/issues.jsonl]
```

If the corpus file is not found, the command exits with an error message and exit code 1.

### JSON output example

```json
{
  "title": "fix nil pointer in dispatcher",
  "type": {
    "value": "bug",
    "confidence": 0.91,
    "status": "ok",
    "neighbors": [
      {"label": "bug", "score": 0.93, "metadata": "sdplab-001"},
      {"label": "bug", "score": 0.88, "metadata": "sdplab-004"}
    ]
  },
  "priority": {
    "value": "P1",
    "confidence": 0.87,
    "status": "ok",
    "neighbors": []
  }
}
```

### Human output example

```
Title: fix nil pointer in dispatcher

Type:     bug      [ok, confidence: 0.91]
Priority: P1       [ok, confidence: 0.87]

Top neighbors (type):
  1. sdplab-001    bug      0.93  "sdplab-001"
  2. sdplab-004    bug      0.88  "sdplab-004"
  3. sdplab-002    feature  0.72  "sdplab-002"
```

## Classifiers

| Classifier       | Input      | Labels           | Threshold | Accuracy               |
|-----------------|------------|------------------|-----------|------------------------|
| WsVerdictMicro  | Diff       | PASS/FAIL        | 0.85      | 100% (deterministic)   |
| BdSeverityMicro | title+desc | P0..P3           | 0.85      | >=80%                  |
| BdTypeMicro     | title+desc | bug/task/feature | 0.80      | >=85%                  |
| RoutingColdStart| title+desc | capability hint  | 0.80      | >=75%                  |

## Corpus format

Classifiers are trained from `.beads/issues.jsonl`. Each line is a JSON object:

```json
{
  "_type": "issue",
  "id": "sdplab-001",
  "title": "fix nil pointer in dispatcher",
  "description": "crash on startup when ws is empty",
  "status": "closed",
  "priority": "P1",
  "issue_type": "bug",
  "created_at": "2025-01-01T00:00:00Z"
}
```

Only `status == "closed"` issues with non-empty `priority` and `issue_type != "epic"` are used.
The corpus is split: last 30 issues go to eval; the rest to train.
When fewer than 30 issues exist, all are used for both training and evaluation.

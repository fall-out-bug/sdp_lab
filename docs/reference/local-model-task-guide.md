# Local Model Task Guide (F133)

> When to route to local (Ollama) vs cloud, and how to split workstreams for local execution.

## Setup

| Component | Value |
|-----------|-------|
| Server | Ollama 0.18.0 at `http://localhost:11434` |
| Primary model | `qwen2.5-coder:7b` (4.7 GB) |
| Fallback model | `deepseek-coder:6.7b` (3.8 GB) |
| Hardware | 24 GB RAM, Apple Metal 4 |

## Benchmark Results (2026-04-18)

| Model | Size | Speed | Time for ~100 tokens | Code Quality | JSON streaming |
|-------|------|-------|----------------------|--------------|----------------|
| `qwen2.5-coder:7b` | 4.7 GB | ~5–6 t/s | ~20s | ✅ Correct Go, follows conventions | `stream:false` broken (control chars) → use `stream:true` |
| `deepseek-coder:6.7b` | 3.8 GB | ~7–8 t/s | ~33s | ⚠️ Occasional missing `return`, extra prose | Same issue |

**Observation:** Both models produce valid code when prompted clearly with minimal context (512 tokens).
Context following is good: correct package names, `strings.Contains` over manual loops, lowercase conv.

## Cloud vs Local Comparison

| Criterion | Cloud (claude-sonnet-4-6) | qwen2.5-coder:7b | deepseek-coder:6.7b |
|-----------|--------------------------|-----------------|---------------------|
| **Response time** | 2–5 s | 15–30 s | 25–40 s |
| **Code quality** | Excellent | Good (compilable) | Fair (occasional bugs) |
| **Context understanding** | Full project (128k) | Prompt-only (4k) | Prompt-only (2k) |
| **Multi-file changes** | ✅ Full support | ❌ Single function only | ❌ Single function only |
| **Architecture decisions** | ✅ With full context | ❌ No | ❌ No |
| **Offline / no-internet** | ❌ Requires network | ✅ Fully offline | ✅ Fully offline |
| **Cost** | ~$0.003/1k tokens | $0 | $0 |
| **Privacy** | Data leaves machine | ✅ Air-gapped | ✅ Air-gapped |
| **System prompt rules** | Full go-patterns.md | Condensed 1-page prompt | Condensed 1-page prompt |

## When Local Is Sufficient

✅ Use local model when the task is:

| Task type | Example |
|-----------|---------|
| Function stub | `NewFoo(cfg Config) (*Foo, error)` body from signature |
| Test stub | Table-driven test for a given function signature |
| Simple refactor | Rename field in one struct |
| Boilerplate | Implement `io.Reader` or `fmt.Stringer` interface |
| Single utility function | `IsLowComplexity(ws string) bool` |
| Error sentinel | Add `var ErrFoo = errors.New("foo")` |

## When Cloud Is Required

❌ Use cloud when:

| Reason | Example |
|--------|---------|
| Cross-file context | Tracing call chain A → B → C across packages |
| Architecture decisions | Should this be a FSM or a simple state var? |
| Multi-package refactor | Rename interface in 5 packages |
| Debug reasoning | "Why does this goroutine leak?" |
| Reading codebase | "Where is X configured?" |
| Generating tests for complex logic | FSM with 4 states and concurrent access |

## WS Decomposition for Local Models

A standard SDP workstream (WS) is typically scoped to one feature at the interface level.
**For local model execution, split to function-level tasks:**

```
Standard WS: "Add rate limiting to ModelGateway"
→ Local-friendly sub-tasks:
  1. Write RateLimiter struct with mu sync.Mutex + tokens int fields
  2. Write NewRateLimiter(rate int) *RateLimiter constructor
  3. Write Allow(ctx context.Context) bool method
  4. Write test stub: TestRateLimiter_Allow table-driven
```

**Rule of thumb:** If the task fits in ≤ 512 tokens of context (prompt + code), local can handle it.
If it needs more context to answer correctly, use cloud.

## Classification Heuristics (for dispatch.Classify)

```go
// Low complexity → local model candidate
keywords := []string{"stub", "boilerplate", "test", "add_field", "rename", "simple", "implement interface"}
```

```
Complexity: low   + RequiredCap: coding   → ollama/qwen2.5-coder:7b
Complexity: medium + RequiredCap: coding   → cloud/coding model
Complexity: *      + RequiredCap: reasoning → cloud/reasoning model
```

## Generation Parameters for Code (local)

```json
{
  "temperature": 0.1,
  "top_p": 0.9,
  "num_ctx": 512,
  "stop": ["```\n\n"]
}
```

Low temperature (0.1) is critical for deterministic, convention-following code generation.
Larger `num_ctx` improves quality at cost of speed; 512–1024 is the sweet spot for function-level tasks.

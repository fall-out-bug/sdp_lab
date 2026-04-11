# Council R6 Plan Review Synthesis

## Council Composition
| Role | Model | Verdict |
|------|-------|---------|
| Critic | Gemini 3.1 Pro | REVISE |
| Technician | DeepSeek V3.2 | REVISE |
| Philosopher | Kimi K2.5 | REVISE |
| Pragmatist | MiniMax M2.7 | REVISE |
| Engineer | Xiaomi MiMo V2 Pro | APPROVE* |

**Consensus: 5/5 agree on approach, 4/5 request specific revisions.**

## Key Decisions

### Fix 1: Import-Graph Direction (NOT POM Parsing)
- **Pragmatist's insight**: POM `<dependency>` parsing is a tar pit (properties, parent POMs, BOM)
- **Alternative**: Use import-graph directionality — if cluster A imports from B, A→B
- **All 5 agreed**: This is the correct MVP approach
- **Critic's caveat**: Must add `src/main/scala/` to javaPackageName for module mapping to work

### Fix 2: Scala with Explicit Handling (NOT "Same as Java")
- **Philosopher**: Scala has traits, objects, implicits — ontologically different
- **Technician**: Regex passes 5/5 test patterns but needs rename-aware expansion
- **Critic**: Multiline brace imports need line-accumulation logic
- **Consensus**: 80% Scala import coverage is acceptable for MVP

### Fix 4: Phantom Filtering with Maven Module Check
- **Technician**: Triple-conjunction filter is too aggressive
- **Add**: "OR is in Maven module list" to preserve resource-only modules
- **All agreed**: Low risk, clear boundaries

### Fix 5: Deferred
- **Pragmatist**: Cosmetic — Python circular deps are a labeling issue
- **All agreed**: Deferrable without impacting 3.5 target

### Line Estimate: ~235 (not 160)
- Engineer: ~300 (conservative)
- Technician: ~200-230 (realistic)
- Plan: ~235 (with council revisions)

## Target: 3.5/5
Critic's projection: D1=4.0, D2=3.0, D3=3.5, D4=3.5, D5=2.5, D6=3.5, D7=4.0, D8=3.0 → avg=3.38
Realistic range: 3.25-3.5 depending on Scala brace import coverage.

## Additional Items (Not in Plan)
- spark-rpc should be classified as internal, not external (5-line fix in pipeline.go)
- Test fixture should include .scala files for Fix 2 validation

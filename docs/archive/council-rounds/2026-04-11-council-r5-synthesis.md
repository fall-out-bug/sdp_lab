# LLM Council Round 5 Synthesis: Ground-Truth Validation on Apache Spark

## Council Composition
| Role | Model | Vote |
|------|-------|------|
| Critic | Gemini 3.1 Pro | **VETO** (~2.25 avg) |
| Philosopher | Kimi K2.5 | **VETO** (~2.50 avg) |
| Pragmatist | MiniMax M2.7 | **VETO** (~2.38 avg) |

**Consensus: 3/3 VETO. Average score: ~2.38/5 (down from R4: 4.15)**

## Dimension Ratings
| Dimension | Critic | Philosopher | Pragmatist | Avg |
|-----------|--------|-------------|------------|-----|
| D1 Factual Accuracy | 3 | 3 | 3 | 3.0 |
| D2 Completeness | 2 | 2 | 2 | 2.0 |
| D3 Cluster Quality | 2 | 3 | 2 | 2.3 |
| D4 False Positives | 3 | 3 | 3 | 3.0 |
| D5 Runtime Coupling | 2 | 2 | 2 | 2.0 |
| D6 Architecture Fidelity | 2 | 2 | 2 | 2.0 |
| D7 Relationship Accuracy | 2 | 3 | 3 | 2.7 |
| D8 Practical Value | 2 | 2 | 2 | 2.0 |

## Critical Issues (All 3 Council Members Agree)

### P0: Relationship Direction Errors
| Reported | Correct | Impact |
|----------|---------|--------|
| `catalyst → hive` | `hive → catalyst` | Inverts Catalyst's deliberate independence |
| `core → hive` | `hive → core` | Inverts foundational dependency |
| `core → hive-thriftserver` | `hive-thriftserver → core` | Same inversion |
| `api → catalyst` | `catalyst → api` (or api is leaf) | Misrepresents API layer |

**Root cause**: Maven `<module>` relationships are bidirectional in POM — the tool creates edges between all modules that co-occur, not respecting actual `<dependency>` direction.

### P1: sql/core Invisible
- Detected as 1-package cluster with 0 edges
- Reality: sql/core is the **largest and most important module** in modern Spark
- Contains DataFrame/Dataset execution, Whole-Stage CodeGen, Structured Streaming
- The adaptive clustering correctly mapped it to `spark-sql-core` but only captured 1 Java package dir
- Most of sql/core's code is **Scala** (`.scala` files), not Java — the Java extractor ignores Scala

### P2: Python Monolith (914 packages)
- All PySpark dumped into one cluster `spark-3.5.7`
- PySpark has clear subsystems: `pyspark.sql`, `pyspark.ml`, `pyspark.streaming`, `pyspark.mllib`
- The `pyPathToModule()` prefixes paths with `spark-3.5.7` (repo subdirectory)
- Python clustering doesn't respect PySpark's real package boundaries

### P3: Phantom Modules
- `network-yarn` — no such Maven module exists; likely `common/network-yarn` misclassified
- `spark-runtime` — artifact of runtime coupling analysis, not a real module
- `spark-rm`, `master`, `worker` — these are deployment classes, not Maven modules

### P4: 559 Python Circular Deps Misleading
- Maven dependencies are a strict DAG (0 cycles)
- Python circular imports are a packaging idiom, not an architectural flaw
- Conflating these alarms developers without cause

## Non-Critical Issues (1-2 members)

| Issue | Severity | Source |
|-------|----------|--------|
| `spark-rpc` classified as external system (it's internal) | Medium | Philosopher, Pragmatist |
| Spark Connect gRPC not detected | Medium | Philosopher, Pragmatist |
| Py4J control-plane vs data-plane not distinguished | Low | Philosopher |
| `postgres`/`mysql` as external systems (they're JDBC examples) | Low | Critic |
| HDFS/S3 missing from external systems | Low | Critic, Philosopher |
| Driver-Executor model invisible (epistemological limit) | Design | All (acknowledged as static analysis boundary) |

## Score Progression

| Round | Avg | Status | Focus |
|-------|-----|--------|-------|
| R1 | 1.45 | 5/5 VETO | Fatal extraction errors |
| R2 | 3.30 | 3/5 VETO | Consistency gaps |
| R3 | 3.00 | 3/4 VETO | Edge attribution bugs |
| R4 | 4.15 | 3/3 CONVERGE | Fixed extraction bugs |
| **R5** | **2.38** | **3/3 VETO** | **Ground-truth validation exposed semantic errors** |

## Key Insight

R4 converged on **structural correctness** (edges count, no bidirectional Maven deps, artifact filtering). R5 exposed that structural correctness ≠ architectural fidelity. The tool correctly counts edges and clusters packages, but:
1. **Direction matters** — `A→B` vs `B→A` is the difference between correct and inverted architecture
2. **Scala is invisible** — Spark is ~70% Scala; the Java extractor misses `.scala` files entirely
3. **Maven modules ≠ Maven dependencies** — `<modules>` lists children; `<dependencies>` lists actual compile-time deps
4. **Package clusters ≠ architectural layers** — import graph fragments don't reconstruct layered architecture

## Required Fixes for R6

| Priority | Fix | Root Cause | Effort |
|----------|-----|-----------|--------|
| P0 | Parse `<dependency>` from POM instead of `<module>` co-occurrence | Maven relationship direction | Medium |
| P1 | Add Scala source support (`.scala` → same as `.java` for imports) | 70% of Spark code invisible | Medium |
| P2 | Fix Python path prefix stripping (remove repo subdirectory) | PySpark monolith cluster | Small |
| P3 | Filter phantom containers (no source files → drop) | spark-rm, network-yarn | Small |
| P4 | Separate Python circular deps from Maven cycles in output | Misleading alarms | Small |

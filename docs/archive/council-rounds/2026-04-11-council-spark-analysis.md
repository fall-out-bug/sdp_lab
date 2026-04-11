# LLM Council Round 1 Synthesis: AI Architect on Apache Spark

## Council Composition
| Role | Model | Vote |
|------|-------|------|
| Critic | Gemini 3.1 Pro | **VETO** (D1, D3, D6, D8) |
| Technician | DeepSeek v3.2 | **VETO** (serialization + pipeline) |
| Philosopher | Claude Sonnet 4 | **VETO** (Model Construction) |
| Pragmatist | Grok 4 | **CONVERGE** (unacceptable but fixable) |
| Architect | GPT-4.1 | **VETO** (System Design) |

**Consensus: 4/5 VETO, 1 CONVERGE. Average score: 1.45/5**

## Dimension Ratings
| Dimension | Avg | Critic | Tech | Phil | Prag | Arch |
|-----------|-----|--------|------|------|------|------|
| D1 Language | 1.0 | 1 | 1 | 1 | 1 | 1 |
| D2 Containers | 1.4 | 1 | 2 | 2 | 1 | 1 |
| D3 Relationships | 1.0 | 1 | 1 | 1 | 1 | 1 |
| D4 Import Graph | 1.4 | 1 | 1 | 2 | 2 | 1 |
| D5 External Systems | 3.4 | 3 | 4 | 4 | 4 | 3 |
| D6 System Naming | 1.2 | 1 | 2 | 1 | 1 | 1 |
| D7 SQL False Pos | 1.8 | 2 | 1 | 2 | 2 | 2 |
| D8 Overall | 1.0 | 1 | 1 | 1 | 1 | 1 |

## ROOT CAUSES (Council consensus)

### RC1: 30s timeout catastrophically inadequate for large repos
- Java adapter opens every .java file individually -> times out on 26K files
- This single timeout causes: empty language detection, empty clusters, empty CI data
- **Fix**: Increase default to 120s or make timeout proportional to file count

### RC2: No cross-extractor synthesis
- FileTreeExtractor has perfect ExtCounts data (.scala, .java, .py)
- No code path maps ExtCounts -> LanguageInfo
- No extractor sets CodebaseProfile.Name (pom.xml name, README heading)
- Each adapter is isolated; assembler doesn't synthesize cross-cutting insights

### RC3: Import graph -> C4 model bridge is lossy
- ImportGraph.Nodes/Edges stored as int counts (design choice)
- Actual topology lives in canonical Edges []StructuralEdge (never populated by adapters)
- JavaAdapter creates clusters but doesn't populate name/node_count/edge_count
- Relationship inference reads from empty clusters -> 0 relationships

### RC4: Web-service architecture bias
- Container detection assumes Dockerfile = C4 Container (CI images included)
- Relationship inference assumes docker-compose depends_on (Spark uses Maven modules)
- No Maven module dependency inference for inter-module relationships
- No Scala/SBT support in adapter registry

## P0 Issues

### P0-1: Language detection empty
- File: assembler.go computeMetrics() + adapters.go
- Cause: 30s timeout kills JavaAdapter on large repos; no FileTree ExtCounts fallback
- Fix: (1) Add language inference from ExtCounts in computeMetrics; (2) Increase timeout

### P0-2: 0 relationships
- File: c4/relationship.go inferRelationships()
- Cause: Import clusters have empty fields -> buildPackageToContainerMap returns no matches
- Fix: (1) Fix JavaAdapter cluster population; (2) Add Maven module dependency edges

### P0-3: System name "unknown-system"
- File: pipeline.go BuildReferenceModelFromProfile() line 258
- Cause: CodebaseProfile.Name never set by any extractor
- Fix: Read pom.xml name, README.md first heading, or directory basename

### P0-4: Import clusters empty
- File: adapters.go JavaAdapter convertJavaResult()
- Cause: Creates cluster objects but doesn't populate fields (name, node_count, edge_count)
- Fix: Group PackageImports by first 2 path segments, populate cluster metadata

## P1 Issues

### P1-1: CI containers mixed with deploy units
- File: pipeline.go lines 286-337, c4/generator.go line 116
- Cause: Every Dockerfile becomes C4 Container regardless of purpose
- Fix: Classify by directory (.github/, ci/) and content (no EXPOSE = CI)

### P1-2: Dual container creation paths
- File: pipeline.go BuildReferenceModelFromProfile() + c4/generator.go Generate()
- Cause: Two competing paths both create containers -> 58 total
- Fix: Consolidate into single path

### P1-3: SQL test fixtures as real tables
- File: sql_extract.go Extract()
- Cause: No path-based filtering (test/, fixtures/, src/test/)
- Fix: Skip .sql files under test directories

## P2 Issues

### P2-1: Missing big-data ecosystem signals
- HDFS, YARN, Mesos, Hive Metastore, ZooKeeper not in depSystemMap
- Fix: Add hadoop/zookeeper/mesos/yarn patterns

### P2-2: No Scala/SBT support
- build.sbt, project/plugins.sbt not parsed
- .scala files not detected as Scala language
- Fix: Add ScalaAdapter or extend JavaAdapter

## Prioritized Fix Plan

| # | Fix | Effort | Impact |
|---|-----|--------|--------|
| 1 | Language fallback from ExtCounts | ~20 lines | Fixes D1 |
| 2 | System name from pom.xml/README/dir | ~15 lines | Fixes D6 |
| 3 | Fix JavaAdapter cluster population | ~30 lines | Fixes D4, enables D3 |
| 4 | Relationship inference from module boundaries | ~25 lines | Fixes D3 |
| 5 | CI container filtering | ~20 lines | Fixes D2 |
| 6 | SQL test path filtering | ~10 lines | Fixes D7 |
| 7 | Increase default timeout to 120s | ~5 lines | Fixes cascading failures |
| 8 | Add big-data ecosystem signals | ~10 lines | Improves D5 |
| 9 | Add Scala/SBT support | ~50 lines | Improves D1, D4 |
| **Total** | **~185 lines** | | |

Council quote: "The structural pipeline is sound -- extractors run, fragments merge, C4 is generated. The problems are all addressable with targeted fixes totaling approximately 90-100 lines of code." -- Pragmatist (Grok 4)

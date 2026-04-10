# C4 Generation Algorithm Specification

Date: 2026-04-10
Status: Draft
Owner: F105 (AI Architect)
Depends on: WS-02 (Core Types), WS-10 (C4 Model Generation)

---

## Overview

This document specifies the algorithm for generating C4 architecture diagrams from
extractor outputs. The generation is split into two phases:

1. **Phase 1 — Deterministic Node/Edge Creation**: Pure Go, no LLM. Builds the
   structural graph from extractor signals.
2. **Phase 2 — LLM Semantic Enrichment**: Adds descriptions, technology tags, and
   business purpose labels. Never modifies structure.

---

## Section 1: ReferenceModel Graph Schema

### Node Types

```
System          — top-level node, one per repo root
  id:           string (repo root path hash)
  name:         string (repo basename or configured name)

Container       — independently deployable unit
  id:           string (canonical path to deploy unit root)
  name:         string (service/package name)
  containerType: enum(Service, Database, MessageQueue, Cache, CDN, BlobStore, Function)

Component       — logical module within a container
  id:           string (container_id + "/" + component slug)
  name:         string (derived from package/dir name)
  language:     string (Go, Python, Java, TypeScript, SQL, etc.)

CodeElement     — fine-grained unit (function, type, interface)
  id:           string (component_id + "/" + element name)
  name:         string (symbol name)
  kind:         enum(Function, Interface, Struct, Class, Type, Enum)
```

### Edge Types

```
Contains        — parent/child ownership (System->Container, Container->Component)
  source:       System | Container
  target:       Container | Component
  invariant:    child has exactly one parent

Uses            — runtime or compile-time dependency
  source:       Component | Container
  target:       Component | Container
  protocol:     enum(HTTPRequest, GRPC, Import, FunctionCall, MessageQueue, SharedLibrary)

Implements      — contract fulfillment (API spec -> handler)
  source:       Component
  target:       CodeElement (interface or spec definition)
  specType:     enum(OpenAPI, Proto, GraphQL, AsyncAPI)

PersistsTo      — data storage dependency
  source:       Container
  target:       Container (of type Database)
  schema:       string[] (table or collection names)

Exposes         — public API surface
  source:       Component
  target:       CodeElement (route, handler, endpoint)
  path:         string (URL path or gRPC method)
  method:       string (HTTP verb or gRPC method)
```

### Graph Invariants

1. Every Container has at least one Component
2. No cycles in Contains edges (tree structure)
3. Uses edges between Containers must be cross-service (never self-referential)
4. Uses edges between Components within the same Container are intra-service
5. Every PersistsTo target is a Container of type Database
6. System node is the single root; exactly one System per analysis run

---

## Section 2: Deterministic Node Creation (Phase 1)

No LLM calls. Pure Go functions operating on extractor outputs.

### 2.1 System Node

```
input:  repo root path
output: System node

algorithm:
  1. id = sha256(repo_root_path)[:16]
  2. name = basename(repo_root_path)
  3. Detect monorepo: check for multiple deploy units at root
  4. If monorepo, set system.monorepo = true
```

### 2.2 Container Nodes

```
input:  InfraExtractor output + DependencyManifestParser output
output: []Container

detection rules (evaluated in order, first match wins):

  Dockerfile:
    - Each Dockerfile at repo root or service/*/Dockerfile -> Container of type Service
    - Name = directory containing Dockerfile

  docker-compose:
    - Each service: block -> Container of type Service
    - Name = service key from compose file
    - If image: field references another container -> Uses edge candidate

  Kubernetes:
    - Each Deployment resource -> Container of type Service
    - Name = metadata.name from deployment
    - Containers sharing a label selector are grouped

  Go cmd/ directories:
    - Each cmd/<name>/ directory with main.go -> Container of type Service
    - Name = <name> from directory

  npm/yarn/pnpm workspaces:
    - Each workspace package in workspaces array -> Container of type Service
    - Name = package name from workspace package.json

  Maven multi-module:
    - Each <module> in parent pom.xml -> Container of type Service
    - Name = module artifactId

  Gradle multi-project:
    - Each include in settings.gradle -> Container of type Service
    - Name = project name

  Database detection:
    - docker-compose service with image: postgres|mysql|mongo|redis|elasticsearch
      -> Container of type Database
    - Terraform resource: aws_db_instance, aws_rds_cluster, google_sql_database_instance
      -> Container of type Database
    - If no explicit DB infra, infer from SQLExtractor: each schema/database -> Database Container

  Message queue detection:
    - docker-compose service with image: rabbitmq|kafka|nats|sqs
      -> Container of type MessageQueue
    - Terraform resource: aws_sqs_queue, aws_sns_topic
      -> Container of type MessageQueue
```

### 2.3 Component Nodes

```
input:  Language extractor outputs (Go, Python, Java, TS/JS, SQL)
output: []Component

algorithm per Container:
  1. Collect all import graph edges within this container
  2. Build adjacency list from import statements
  3. Cluster using connected components with threshold >= 3 internal edges
  4. Each cluster -> Component
  5. If no clustering applies (few files), each top-level package/dir -> Component

  fallback (when import graph is sparse):
    - Group by directory: files in same directory -> one Component
    - Override: if directory has >20 files, split by naming convention (controller, service, model, etc.)

  naming:
    - Use directory name if cluster maps to a single directory
    - Use dominant package/import path last segment otherwise
    - Lowercase, hyphens to underscores
```

### 2.4 CodeElement Nodes (Deferred)

```
  CodeElement creation is deferred to L4 on-demand generation.
  Phase 1 creates CodeElement nodes only for:
    - Interface types detected by Go extractor (go/types.Info)
    - Spring @Bean methods
    - exported function signatures that appear in >3 Uses edges (API surfaces)
```

---

## Section 3: Deterministic Edge Creation

### 3.1 Contains Edges

```
  System --Contains--> Container    (one per container)
  Container --Contains--> Component (one per component)

  algorithm:
    for each container:
      create Contains(System, Container)
      for each component in container:
        create Contains(Container, Component)
```

### 3.2 Uses Edges (Component-to-Component)

```
  input: import graph from language extractors

  algorithm:
    1. For each import statement (source_file -> imported_package):
       a. Resolve source_file to its Component
       b. Resolve imported_package to its Component (may be external -> skip)
       c. If source != target and both are internal:
          create Uses(source_component, target_component, protocol=Import)
       d. Deduplicate: merge edges with same (source, target, protocol) into one

    2. For cross-container imports:
       If source_component.container != target_component.container:
         Also create Uses(source_container, target_container, protocol=Import)
         (this becomes a cross-service dependency)

  deduplication:
    - Same (source, target, protocol) -> increment weight counter on existing edge
    - Edges with weight >= 3 are "strong" dependencies (highlighted in diagrams)
```

### 3.3 Uses Edges (Container-to-Container)

```
  input: HTTP handler analysis, gRPC service references

  HTTP detection:
    1. For each route handler found by framework detection (Flask, Spring, Express, etc.):
       a. Resolve handler to its Container
       b. If handler calls an internal URL (detected from config/constants):
          create Uses(source_container, target_container, protocol=HTTPRequest)
    2. OpenAPI spec references:
       For each $ref in OpenAPI specs that points to another service:
         create Uses(source_container, target_container, protocol=HTTPRequest)

  gRPC detection:
    1. For each .proto file with service definition:
       a. Resolve to Container that implements this service
       b. For each client import of generated proto stubs:
          create Uses(client_container, server_container, protocol=GRPC)
```

### 3.4 PersistsTo Edges

```
  input: SQLExtractor output, ORM model correlation

  algorithm:
    1. For each table/schema in SQLExtractor output:
       a. Resolve to a Database Container (created in Section 2.2)
       b. For each ORM model that maps to this table:
          Resolve ORM model to its Component
          create PersistsTo(component.container, database_container, schema=[tables])
    2. For each migration directory:
       Resolve migration dir to its Container
       If target database is identifiable:
         create PersistsTo(container, database_container, schema=[migrated_tables])
    3. Deduplicate: merge PersistsTo edges with same (source, target) pair

  database container resolution:
    - Explicit: docker-compose/terraform references
    - Implicit: connection string patterns in config files
    - Fallback: single "default-db" Container if no explicit DB found
```

### 3.5 Implements Edges

```
  input: SpecInventoryScanner output

  algorithm:
    1. For each OpenAPI spec:
       a. Find Container(s) with handlers matching spec paths
       b. create Implements(component, spec_element, specType=OpenAPI)
    2. For each .proto service definition:
       a. Find Container(s) that import generated proto stubs
       b. create Implements(component, proto_service, specType=Proto)
    3. For each GraphQL schema:
       a. Find resolvers matching schema types
       b. create Implements(component, schema_type, specType=GraphQL)
```

### 3.6 Exposes Edges

```
  input: Framework detection output (route tables)

  algorithm:
    1. For each detected route (method + path):
       a. Resolve handler to its Component
       b. create Exposes(component, handler_element, path=path, method=method)
    2. For each gRPC service method:
       a. Resolve to its Component
       b. create Exposes(component, method_element, path="/package.Service/Method")
```

---

## Section 4: LLM Semantic Enrichment (Phase 2)

Phase 2 receives the deterministic graph from Phase 1 and enriches it with
semantic annotations. It MUST NOT modify the graph structure.

### 4.1 Input Contract

```
  input: DeterministicGraph (nodes + edges from Phase 1)
  output: EnrichedGraph (same nodes + edges, with added annotations)

  strict rule: node_count(phase2_output) == node_count(phase1_output)
  strict rule: edge_count(phase2_output) == edge_count(phase1_output)
```

### 4.2 Enrichment Fields

Phase 2 adds exactly these fields to existing nodes and edges:

**Nodes (Container, Component):**
- `description`: string — 1-2 sentence purpose statement
- `technologyTags`: string[] — e.g., ["Go", "Gin", "PostgreSQL", "Docker"]
- `businessPurpose`: string — business domain label (e.g., "Payment Processing", "User Auth")

**Edges (Uses, PersistsTo):**
- `description`: string — nature of the dependency
- `dataFlow`: string — what data flows across this edge (sanitized, no secrets)

### 4.3 LLM Prompt Template

```
You are an architecture analyst. You receive a JSON graph of a codebase's structure
(nodes and edges). Your job is to add semantic annotations to each element.

STRICT RULES:
- Do NOT add, remove, or merge any nodes
- Do NOT add, remove, or merge any edges
- Do NOT change any id, name, or structural field
- ONLY add: description, technologyTags, businessPurpose for nodes
- ONLY add: description, dataFlow for edges

INPUT GRAPH:
{{ .GraphJSON }}

OUTPUT FORMAT:
Return the same JSON with these additional fields populated:
- Nodes: add "description", "technologyTags", "businessPurpose"
- Edges: add "description", "dataFlow"

Keep descriptions concise (1-2 sentences). Technology tags should be specific
library/framework names, not generic categories. Business purpose should be a
short domain label.

Respond with valid JSON only. No markdown fences.
```

### 4.4 Token Budget Management

```
  Tier 1 enrichment (System + Containers only):
    - ~500 tokens input + ~500 tokens output per container
    - Total: ~2K tokens for 4-container system

  Tier 2 enrichment (Components):
    - ~200 tokens input + ~200 tokens output per component
    - Batch up to 20 components per LLM call to amortize overhead
    - Total: ~5-10K tokens for 30-component system

  Tier 3 enrichment (edges):
    - ~100 tokens per edge
    - Batch edges by source container
    - Total: ~2-5K tokens for 50-edge system
```

### 4.5 Validation Guard

After LLM response parsing, validate:

```
  func validateEnrichment(original Graph, enriched Graph) error:
    if len(original.Nodes) != len(enriched.Nodes):
      return error("node count mismatch: LLM added/removed nodes")
    if len(original.Edges) != len(enriched.Edges):
      return error("edge count mismatch: LLM added/removed edges")
    for each original node:
      enriched_node = enriched.getNode(node.ID)
      if enriched_node == nil:
        return error("node %s missing in enriched graph", node.ID)
      if enriched_node.Name != node.Name:
        return error("node %s name changed by LLM", node.ID)
    for each original edge:
      enriched_edge = enriched.getEdge(edge.ID)
      if enriched_edge == nil:
        return error("edge %s missing in enriched graph", edge.ID)
    return nil
```

If validation fails, discard LLM enrichment and use deterministic graph with
placeholder annotations ("[Needs human description]").

---

## Section 5: Mermaid Rendering

### 5.1 L1 System Context Diagram

```mermaid
graph TB
    subgraph System["{{ .SystemName }}"]
        {{ range .Containers }}
        {{ .ID }}["{{ .Name }}<br/><i>{{ .Description }}</i>"]
        {{ end }}
    end
    {{ range .ExternalActors }}
    {{ .ID }}(("{{ .Name }}"))
    {{ end }}
    {{ range .CrossSystemEdges }}
    {{ .Source }} -->|"{{ .Label }}"| {{ .Target }}
    {{ end }}
```

Layout hints:
- `graph TB` (top-bottom) for systems with <=5 containers
- Container nodes use `[]` notation
- External actors use `(())` notation
- Max 10 nodes per L1 diagram

### 5.2 L2 Container Diagram

```mermaid
graph LR
    subgraph System["{{ .SystemName }}"]
        {{ range .Services }}
        {{ .ID }}["{{ .Name }}<br/>{{ .TechTags }}"]
        {{ end }}
        {{ range .Databases }}
        {{ .ID }}[("{{ .Name }}<br/>{{ .DBType }}")]
        {{ end }}
    end
    {{ range .ContainerEdges }}
    {{ .Source }} -->|"{{ .Protocol }}"| {{ .Target }}
    {{ end }}
```

Layout hints:
- `graph LR` (left-right) for container interactions
- Services use `[]`, databases use `[()]` cylinder notation
- Group by network boundary when docker-compose networks are available
- Max 15 nodes per L2 diagram

### 5.3 L3 Component Diagram

```mermaid
graph TB
    subgraph Container["{{ .ContainerName }}"]
        {{ range .Components }}
        {{ .ID }}["{{ .Name }}"]
        {{ end }}
    end
    {{ range .ComponentEdges }}
    {{ .Source }} -.->|"{{ .Weight }} imports"| {{ .Target }}
    {{ end }}
    {{ range .ExternalDeps }}
    {{ .ID }}(["{{ .PackageName }}"])
    {{ end }}
```

Layout hints:
- Dotted arrows for internal dependencies, solid for cross-container
- Edge thickness proportional to weight (import count)
- External deps grouped in a separate subgraph
- Max 20 nodes per L3 diagram (split into sub-diagrams if exceeded)

### 5.4 Fallback for Large Diagrams (>15 nodes)

When a diagram exceeds 15 nodes:

1. Emit the full Mermaid diagram with a comment: `%% WARNING: >15 nodes, layout may be suboptimal`
2. Emit a structured graph data file alongside:

```json
{
  "level": "L2",
  "system": "my-project",
  "nodes": [
    {"id": "api-svc", "type": "Container", "label": "API Service", "tech": ["Go", "Gin"]}
  ],
  "edges": [
    {"source": "api-svc", "target": "user-db", "type": "PersistsTo", "label": "reads/writes users"}
  ],
  "layout_suggestion": "Arrange databases on the right, services in center, external actors on left"
}
```

3. The graph data file can be imported into Excalidraw or draw.io for manual layout.

### 5.5 Output File Locations

```
{target-repo}/.sdp/architecture/c4/
  level-1-system-context.mmd
  level-2-container.mmd
  level-3-{container-name}.mmd    (one per container)
  level-2-container.graph.json    (only if >15 nodes)
  level-3-{container-name}.graph.json  (only if >15 nodes)
```

---

## Section 6: Confidence Scoring

### 6.1 Node Confidence

```
  Container confidence:
    - Dockerfile found:           0.95
    - docker-compose service:     0.95
    - k8s Deployment:             0.90
    - Go cmd/ directory:          0.85
    - npm workspace:              0.85
    - Maven module:               0.80
    - Gradle subproject:          0.80
    - Inferred from dir structure: 0.50

  Component confidence:
    - Import graph cluster >=5 edges:  0.90
    - Import graph cluster 3-4 edges:  0.75
    - Directory-based grouping:        0.60
    - Single file (no clustering):     0.40

  Database Container confidence:
    - docker-compose DB image:    0.95
    - Terraform DB resource:      0.95
    - SQLExtractor schema match:  0.80
    - Connection string inference: 0.60
    - Fallback "default-db":      0.30
```

### 6.2 Edge Confidence

```
  Contains edge:
    - Always 1.0 (structural, no inference)

  Uses edge (import-based):
    - Direct import statement:     0.90
    - tsconfig path resolution:    0.80
    - Re-export chain (1 hop):     0.70
    - Re-export chain (>1 hop):    0.50

  Uses edge (HTTP/gRPC):
    - OpenAPI $ref match:          0.90
    - Proto service import:        0.90
    - URL constant in handler:     0.70
    - Service name in config:      0.60

  PersistsTo edge:
    - ORM model -> table match:    0.85
    - Migration -> table match:    0.80
    - Connection string inference: 0.60
    - FK relationship:             0.90

  Implements edge:
    - Spec path == handler path:   0.95
    - Proto method == gRPC method: 0.95
    - Spec path pattern match:     0.75
```

### 6.3 Confidence Aggregation

```
  Diagram confidence = weighted average of all elements:
    node_confidence_avg = mean(all node confidences)
    edge_confidence_avg = mean(all edge confidences)
    diagram_confidence = 0.6 * node_confidence_avg + 0.4 * edge_confidence_avg
```

### 6.4 Low-Confidence Markers

Elements below confidence thresholds are marked for human review:

```
  thresholds:
    - confidence >= 0.80: high confidence, no marker
    - 0.60 <= confidence < 0.80: medium confidence, prefixed with "[AUTO?]"
    - confidence < 0.60: low confidence, prefixed with "[AUTO]" and
                         human_description set to null (not populated by LLM)

  output format in Mermaid:
    high:    api_svc["API Service"]
    medium:  auth_svc["[AUTO?] Auth Service"]
    low:     cron_job["[AUTO] Scheduled Task"]

  output format in JSON:
    {
      "id": "cron-job",
      "name": "Scheduled Task",
      "confidence": 0.45,
      "needs_review": true,
      "human_description": null,
      "auto_description": "Detected from directory structure; no Dockerfile or deploy config found"
    }
```

### 6.5 Review Report

A summary section in the output report lists all low-confidence elements:

```json
{
  "review_required": [
    {
      "element_id": "default-db",
      "element_type": "Container",
      "confidence": 0.30,
      "reason": "No explicit database infrastructure found; inferred from SQL migration files",
      "suggestion": "Add a Dockerfile or docker-compose entry for the database service"
    }
  ],
  "stats": {
    "total_nodes": 12,
    "high_confidence": 8,
    "medium_confidence": 3,
    "low_confidence": 1,
    "overall_confidence": 0.78
  }
}
```

---

## Appendix: Sequence Diagram — Full C4 Generation Flow

```
Extractors --> Assembler: ExtractorOutput[]
Assembler --> C4Generator: CodebaseProfile
C4Generator.Phase1:
  InfraExtractor --> create Container nodes
  LanguageExtractors --> create Component nodes
  ImportGraph --> create Uses edges
  SQLExtractor --> create PersistsTo edges
  SpecScanner --> create Implements edges
  FrameworkDetector --> create Exposes edges
  --> validate invariants
  --> compute confidence scores
C4Generator.Phase2 (if LLM enabled):
  SecurityFilter --> sanitize graph
  --> LLM prompt (enrichment only)
  --> validate enrichment (no structural changes)
  --> merge annotations
C4Generator.Output:
  --> Mermaid templates (L1/L2/L3)
  --> Confidence report
  --> Graph data files (large diagrams)
```

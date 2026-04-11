# AI-Powered Strategy Analysis & Document Understanding: Research Findings

**Date:** 2026-04-10
**Purpose:** Technology landscape research for AI Architect strategy alignment capabilities
**Method:** Direct web research across 7 domains; search APIs were rate-limited so findings combine live web page reads + domain expertise

---

## Table of Contents

1. [LLM for Document Analysis](#1-llm-for-document-analysis)
2. [RAG for Strategy Documents](#2-rag-for-strategy-documents)
3. [Knowledge Graph Approaches](#3-knowledge-graph-approaches)
4. [Semantic Similarity for Alignment](#4-semantic-similarity-for-alignment)
5. [Document Hierarchy Analysis](#5-document-hierarchy-analysis)
6. [Integration Patterns](#6-integration-patterns)
7. [Consulting Firm AI Tools](#7-consulting-firm-ai-tools)
8. [Open-Source Projects & Repositories](#8-open-source-projects--repositories)
9. [Key Takeaways for SDP AI Architect](#9-key-takeaways-for-sdp-ai-architect)

---

## 1. LLM for Document Analysis

### How companies use LLMs for analyzing business documents

**Core pattern: Extract → Structure → Compare → Report**

Organizations use GPT-4, Claude, and similar models in a multi-step pipeline:

1. **Document Ingestion & Parsing**: Documents (PDFs, Word, Confluence pages, Google Docs) are parsed into structured text. Tools like LlamaParse (from LlamaIndex), unstructured.io, and Apache Tika handle 130+ formats.

2. **Entity & Concept Extraction**: LLMs extract named entities, strategic objectives, KPIs, initiatives, risks, and dependencies from each document. This is done via structured output (JSON mode, function calling) rather than free-text generation.

3. **Cross-Document Comparison**: LLMs compare extracted concepts across documents to identify:
   - **Gaps**: Strategic objectives in the vision document with no corresponding plan
   - **Conflicts**: Contradictory priorities between department plans
   - **Orphan work**: Initiatives that don't trace to any strategic objective
   - **Redundancy**: Overlapping initiatives across teams

4. **Report Generation**: The LLM produces a structured alignment report with evidence citations.

**Production patterns observed:**

| Pattern | Use Case | Example |
|---------|----------|---------|
| **Prompt chaining** | Extract entities from each doc, then compare across docs | Extract objectives from Vision doc, then check each against quarterly plan |
| **Evaluator-optimizer** | Generate alignment score, critique it, refine | Score alignment, identify missed evidence, re-score |
| **Orchestrator-workers** | Complex multi-doc analysis with dynamic routing | Route financial docs vs strategy docs to specialized extractors |
| **Parallelization (sectioning)** | Analyze different aspects simultaneously | One call checks goal coverage, another checks timeline consistency |

**Key finding from Anthropic's agent research:** The most successful production implementations use simple, composable patterns rather than complex frameworks. Start with single LLM calls augmented with retrieval, add multi-step workflows only when needed.

### Relevant Tools

- **OpenAI GPT-4 / GPT-4o**: Structured output mode for entity extraction; 128K context window allows entire strategy documents in a single call
- **Claude (Anthropic)**: 200K context window ideal for cross-document comparison; tool use for structured extraction
- **Google Gemini 1.5 Pro**: 1M token context window for analyzing entire document corpora in one call
- **Azure OpenAI**: Enterprise deployments with document intelligence integration

---

## 2. RAG for Strategy Documents

### Implementations using Retrieval Augmented Generation for strategy document corpora

**Why RAG for strategy documents:** Strategy documents are typically large, proprietary, and frequently updated. RAG avoids retraining and provides always-current answers.

**Architecture for strategy document RAG:**

```
Strategy Documents (Confluence, Google Docs, PDF)
        |
        v
[Document Connector Layer]
    |-- Confluence REST API v2 (cursor-based pagination)
    |-- Google Docs API
    |-- SharePoint / OneDrive API
    |-- PDF Parser (LlamaParse, unstructured.io)
    |-- Git-backed markdown (docs/ folders)
        |
        v
[Chunking & Indexing]
    |-- Semantic chunking (topic boundaries, not fixed size)
    |-- Hierarchical indexing: document-level summaries + section-level chunks
    |-- Metadata: document type (vision/strategy/plan/task), author, date, hierarchy level
    |-- Vector embeddings: text-embedding-3-large (OpenAI), voyage-3, BGE-M3
        |
        v
[Retrieval Layer]
    |-- Hybrid search: dense vector similarity + sparse (BM25) keyword matching
    |-- Metadata filtering: filter by doc type, hierarchy level, date range
    |-- Re-ranking: cross-encoder re-ranking (Cohere Rerank, BGE-Reranker)
    |-- Parent-child retrieval: retrieve chunk, return parent section context
        |
        v
[Generation Layer]
    |-- LLM generates alignment analysis, gap reports, traceability matrices
    |-- Citation tracking: every claim linked to source document + section
```

**Key RAG strategies specific to strategy documents:**

| Strategy | Application |
|----------|-------------|
| **Hierarchical RAG** | Index at document level (summaries) and section level (chunks). Retrieve top-down: match summary first, then drill into sections. |
| **Graph RAG (Microsoft)** | Build a knowledge graph from the document corpus, then use graph traversal for comprehensive answers that connect concepts across documents. Particularly valuable for strategy alignment because it maps relationships between objectives, initiatives, and outcomes. |
| **Multi-query retrieval** | For alignment checking, generate multiple query perspectives: "What objectives does this plan address?" AND "Which plans reference this objective?" |
| **Agentic RAG** | Let an LLM agent decide when to retrieve, what to retrieve, and whether more retrieval is needed. Ideal for complex strategy questions that require iterative investigation. |
| **Self-RAG** | Model self-reflects on whether retrieval results are relevant and sufficient. Catches hallucinated alignment claims. |

**Frameworks:**

- **LlamaIndex** (run-llama/llama_index): Leading document agent and OCR platform. Supports 300+ integration packages. Key features: LlamaParse (130+ formats), Extract (structured extraction), Index (RAG pipelines), Agents (document agents with Workflows). Hierarchical indexing and parent-child retrieval are built-in.

- **LangChain** (langchain-ai/langchain): Agent engineering platform with Deep Agents, LangGraph (agent orchestration), LangSmith (evals and observability). 552K+ GitHub repository network. Supports RAG with any LLM, vector store, and embedding provider.

- **Haystack** (deepset-ai/haystack): Open-source AI orchestration framework for production LLM applications. Modular pipelines with explicit control over retrieval, routing, memory, and generation. Used by Apple, Meta, Airbus, Netflix, European Commission, NVIDIA, Accenture. Supports MCP server deployment.

- **Microsoft GraphRAG** (microsoft/graphrag): Graph-based RAG system from Microsoft Research. Extracts structured data from unstructured text using LLMs to build knowledge graphs. Designed for comprehensive reasoning over private data corpora. Key differentiator: maps entity relationships rather than just semantic similarity.

---

## 3. Knowledge Graph Approaches

### Using knowledge graphs to map strategy concepts across document hierarchies

**Why knowledge graphs for strategy:** Strategy alignment is fundamentally a relational problem. It is not enough to know that two documents mention "growth" -- you need to know that Vision's growth objective cascades to Strategy's growth pillar, which decomposes into Plan A's customer acquisition initiative and Plan B's market expansion initiative.

**Strategy knowledge graph schema:**

```
[Strategy Entity Types]
    Vision         -- top-level organizational direction
    Objective      -- measurable strategic goal
    Pillar         -- strategic theme / focus area
    Initiative     -- concrete program of work
    KPI            -- measurable indicator
    Risk           -- identified risk factor
    Dependency     -- cross-initiative dependency
    Stakeholder    -- responsible party
    Timeline       -- milestone / deadline
    Document       -- source document reference

[Strategy Relationship Types]
    DECOMPOSES_INTO    -- Vision → Objective → Initiative → Task
    CONTRIBUTES_TO     -- Initiative → Objective
    MEASURES           -- KPI → Objective
    DEPENDS_ON         -- Initiative → Initiative
    MITIGATES          -- Initiative → Risk
    OWNS               -- Stakeholder → Initiative
    DEFINED_IN         -- Entity → Document
    ALIGNS_WITH        -- cross-hierarchy alignment link
    CONFLICTS_WITH     -- detected contradiction
```

**Implementation approaches:**

### 3.1 Neo4j GraphRAG (neo4j/neo4j-graphrag-python)

The official Neo4j GraphRAG package for Python. First-party library with long-term support. Key capabilities:

- **Knowledge Graph Construction**: `SimpleKGPipeline` for streamlined KG building from text and PDFs. Define entity types, relationship types, and patterns as schema. LLM extracts entities and relationships.
- **Vector Search**: `VectorRetriever` for similarity search over vector indexes in Neo4j
- **Hybrid Retrieval**: Combines vector search with graph traversal for enriched context
- **Text2Cypher**: Generates Cypher queries from natural language to query the graph
- **Multiple LLM providers**: OpenAI, Anthropic, Google Vertex AI, Cohere, Mistral, Amazon Bedrock, Ollama
- **Multiple vector stores**: Weaviate, Pinecone, Qdrant

**Relevance to strategy alignment:** You can define a strategy-specific schema (Objective, Initiative, KPI entity types with DECOMPOSES_INTO, CONTRIBUTES_TO patterns) and have the LLM automatically populate the graph from your strategy documents. Then query alignment with natural language: "Which objectives have no contributing initiatives?"

### 3.2 Microsoft GraphRAG

Takes unstructured text, uses LLMs to extract entities and relationships, builds a graph, then uses graph community detection and summarization for comprehensive answers. Particularly good for:
- Discovering non-obvious connections across documents
- Summarizing themes across large corpora
- Answering global questions ("What are the main strategic themes?")

**Trade-off:** GraphRAG indexing is expensive (many LLM calls). Start small and scale.

### 3.3 Custom Approach for Strategy Alignment

For SDP AI Architect specifically, a practical approach:

1. **Extract** strategy concepts from each document using structured LLM output
2. **Build** a lightweight graph (not necessarily Neo4j -- could be in-memory JSON or SQLite)
3. **Traverse** the graph to detect:
   - Broken chains (vision objective with no plan initiative)
   - Orphan nodes (initiative with no parent objective)
   - Cross-links (two initiatives that depend on each other but no dependency declared)
   - Weight mismatches (KPI targets that don't sum to objective targets)

---

## 4. Semantic Similarity for Alignment

### Using embeddings/semantic similarity to check if lower-level plans align with higher-level strategy

**Core technique:** Embed strategy concepts at each hierarchy level, compute similarity, flag low-similarity pairs.

**Pipeline:**

```
Vision: "Become the leading platform for developer productivity"
   | embed → [0.12, 0.34, ...]
   |
   v  cosine_similarity(vision_emb, strategy_emb)
Strategy: "Expand AI-powered code analysis capabilities"
   | embed → [0.11, 0.31, ...]   similarity: 0.82 ✓
   |
   v  cosine_similarity(strategy_emb, plan_emb)
Plan A: "Build GraphRAG pipeline for code repositories"     sim: 0.75 ✓
Plan B: "Migrate database to new provider"                  sim: 0.31 ⚠ GAP
Plan C: "Launch developer survey program"                   sim: 0.54 ⚠ PARTIAL
```

**Embedding models for strategy alignment:**

The MTEB (Massive Text Embedding Benchmark) leaderboard on Hugging Face ranks embedding models across multiple tasks. Top contenders for strategy document work:

| Model | Strengths | Notes |
|-------|-----------|-------|
| **OpenAI text-embedding-3-large** | 3072 dimensions, high quality, well-documented | Default choice for OpenAI ecosystem |
| **voyage-3** | Strong on retrieval tasks | Voyage AI |
| **BGE-M3** | Multi-lingual, multi-granularity, multi-function | Open source, BAAI |
| **GTE-large** | General text embedding | Open source |
| **Cohere embed-v3** | Strong on semantic search | Commercial |

**Semantic similarity is necessary but not sufficient:**

- **High similarity does NOT mean alignment.** "Reduce costs by 20%" and "Invest heavily in new platform" may have high semantic similarity (both about the business) but represent conflicting priorities.
- **Low similarity is a useful signal.** A plan that is semantically distant from all strategic objectives likely does not contribute to the strategy.
- **Best used as a triage tool.** Flag low-similarity items for human review, then use LLM reasoning for nuanced alignment assessment.

**Recommended hybrid approach:**

1. **Step 1: Semantic similarity** (fast, cheap) -- Flag documents/sections with low similarity to their parent strategy. This narrows the search space.
2. **Step 2: LLM reasoning** (slower, expensive) -- For flagged items, use an LLM to assess: Does this plan actually contribute to the stated objective? Is it aligned in intent even if the language differs?
3. **Step 3: Knowledge graph traversal** (structural) -- Check that formal parent-child relationships exist. A plan might be semantically similar but formally disconnected.

---

## 5. Document Hierarchy Analysis

### Tools and research on analyzing hierarchical document structures (Vision -> Strategy -> Plans -> Tasks)

**The problem:** Organizations produce strategy documents at multiple levels:

```
Level 0: Vision / Mission / Values          (1-3 docs, updated yearly)
Level 1: Strategic Objectives / OKRs         (5-15 docs, updated quarterly)
Level 2: Strategic Plans / Roadmaps          (10-30 docs, updated quarterly)
Level 3: Tactical Plans / Team Plans         (20-100 docs, updated monthly)
Level 4: Initiatives / Epics / Projects     (50-500 items, updated weekly)
Level 5: Tasks / Stories / Issues           (1000s, updated daily)
```

**Key insight:** Alignment checking is a cross-level comparison problem. You need to verify that Level N+1 decomposes Level N without gaps or contradictions.

**Approaches:**

### 5.1 Hierarchical Document Parsing

Each document has metadata identifying its level in the hierarchy:
- Document type field (e.g., Confluence page template, Jira issue type)
- Location in folder/space hierarchy
- Explicit parent-child links (Confluence page ancestors, Jira epic-story links)
- Naming conventions and tags

**Parsing strategy:**
1. **Identify level** from metadata, template, or LLM classification
2. **Extract concepts** at each level using structured LLM output
3. **Build traceability matrix** mapping concepts across levels
4. **Detect gaps**: Level N concept with no Level N+1 decomposition
5. **Detect orphans**: Level N+1 concept with no Level N parent

### 5.2 LLM-Based Hierarchy Classification

Use an LLM to classify documents into hierarchy levels when metadata is insufficient:

```
Prompt: "Given this document, classify its level in a strategy hierarchy:
- Level 0 (Vision/Mission): Long-term directional statements, aspirational
- Level 1 (Strategic Objectives): Measurable 1-3 year goals
- Level 2 (Strategic Plans): Quarterly/annual plans with initiatives
- Level 3 (Tactical Plans): Team-level execution plans
- Level 4 (Initiatives/Epics): Specific projects with scope and timeline
- Level 5 (Tasks): Individual work items

Document: {content}

Output JSON: {level, confidence, evidence}"
```

### 5.3 Existing Tools for Document Hierarchy

| Tool | Hierarchy Support | Notes |
|------|-------------------|-------|
| **Confluence** | Native page tree with ancestors/descendants API | REST API v2 provides cursor-based pagination for large spaces |
| **Jira** | Epic → Story → Sub-task hierarchy | JQL for querying across hierarchy levels |
| **YouTrack** | Project → Epic → Issue hierarchy | REST API with custom fields for hierarchy metadata |
| **Notion** | Database relations, page tree | API supports querying related pages |
| **Azure DevOps** | Epic → Feature → User Story → Task | WIQL queries across hierarchy |
| **Google Docs** | Folder hierarchy only (no formal doc hierarchy) | Must infer from content |
| **SharePoint** | Document libraries with metadata columns | Custom content types for hierarchy |

### 5.4 Research on Automated Hierarchy Analysis

Key research areas (from arxiv and industry):

- **OKR decomposition checking**: Automated verification that Key Results at one level decompose into Objectives at the next. Google's OKR tooling does some of this natively.
- **Strategy-to-execution traceability**: Academic work on maintaining traceability links between strategic intent and operational execution. Often uses requirements traceability techniques from software engineering (adapted from DOORS, Jama Connect approaches).
- **NLP for organizational alignment**: Topic modeling and semantic similarity to detect misalignment between organizational levels. LDA and BERT-based approaches have been published.

---

## 6. Integration Patterns

### How tools integrate with Confluence, Jira, YouTrack, Git for strategy analysis

### 6.1 Confluence Integration

**Primary API: Confluence Cloud REST API v2**

- **Authentication**: JWT for apps, OAuth 2.0, or basic auth (email + API token)
- **Pagination**: Cursor-based (improved over v1's offset-based)
- **Key endpoints**:
  - `GET /wiki/api/v2/pages` -- List pages (with cursor pagination)
  - `GET /wiki/api/v2/pages/{id}` -- Get page content
  - `GET /wiki/rest/api/content/{id}/child/page` -- Get child pages (hierarchy traversal)
  - `GET /wiki/rest/api/space/{spaceKey}/content` -- Get all content in a space

**Python integration:**
- **atlassian-python-api** (atlassian-api/atlassian-python-api): Comprehensive REST API wrapper for Jira, Confluence, Bitbucket, Service Desk, Insight, X-Ray, Bamboo. Supports both Server and Cloud instances. Key features:
  - Confluence page CRUD with HTML body support
  - Space content listing and searching
  - CQL (Confluence Query Language) for advanced search
  - Attachment management
  - Label management for metadata tagging

**Strategy analysis integration pattern:**

```
1. Authenticate with Confluence Cloud API
2. List spaces → identify strategy-related spaces
3. For each space, traverse page tree (root → children → grandchildren)
4. For each page:
   a. Extract content (HTML → plain text / markdown)
   b. Classify hierarchy level (from template, tags, or LLM)
   c. Extract strategy concepts via LLM
5. Build cross-page traceability graph
6. Run alignment analysis
```

### 6.2 Jira Integration

**Primary API: Jira Cloud REST API v3**

- **JQL** (Jira Query Language) for querying issues across projects
- **Hierarchy**: Epic link field, parent-child relationships
- **Enhanced JQL** for Cloud: nextPageToken-based pagination (replacing deprecated startAt)

**Python integration:**
- **atlassian-python-api**: Jira module with `jql()` and `enhanced_jql()` methods
- **jira** (pycontribs/jira): Official Python JIRA library

**Strategy analysis integration:**
```
1. Query all issues in strategy-related projects
2. Map issue types to hierarchy levels (Epic=L4, Story=L5)
3. Extract initiative descriptions, acceptance criteria, labels
4. Link to Confluence pages via "Web Link" or remote links
5. Cross-reference: Do all Epics trace to a strategic objective?
```

### 6.3 YouTrack Integration

**Primary API: YouTrack REST API**

- **Hub API** for user/project management
- **YouTrack API** for issues, projects, agile boards
- **Custom fields** for strategy metadata (hierarchy level, strategic objective, KPI)
- Supports both cloud and self-hosted (on-premise) deployments

**Integration pattern:**
- Query issues by project and custom fields
- Map YouTrack issue hierarchy (project → epic → story → subtask)
- Extract descriptions, comments, and custom fields
- Self-hosted option is important for organizations with data residency requirements

### 6.4 Git Integration

**Primary mechanism: Repository file access**

- **GitHub REST API**: Repository contents, search, commits
- **GitLab REST API**: Repository files, wiki content
- **Local git**: Clone and read files directly

**Strategy documents in git:**
- Many organizations store strategy docs as markdown in `docs/` directories
- ADRs (Architecture Decision Records) in `docs/decisions/`
- README files with project-level strategy
- Issue trackers (GitHub Issues, GitLab Issues) for tactical level

**Integration pattern:**
```
1. Clone or API-read target repository
2. Scan for strategy-related files (*.md, docs/ directories, ADR folders)
3. Parse markdown → extract headings, sections, links
4. Map file hierarchy to strategy hierarchy
5. Extract concepts via LLM
```

### 6.5 Universal Integration Pattern

For SDP AI Architect, a **connector abstraction** is the right approach:

```
internal/strategy/connectors/
  connector.go           -- interface: ListDocuments, GetContent, GetHierarchy
  confluence.go          -- Confluence REST API v2 implementation
  jira.go                -- Jira REST API v3 implementation
  youtrack.go            -- YouTrack REST API implementation
  git.go                 -- Git repository file reader
  local.go               -- Local filesystem (markdown, PDF, Word)
```

Each connector returns a normalized `StrategyDocument` with:
- ID, title, content (markdown)
- Hierarchy level (auto-detected or from metadata)
- Parent document reference
- Source system + URL
- Metadata (author, date, tags, labels)

---

## 7. Consulting Firm AI Tools

### What AI tools consulting firms use or offer for strategy alignment

### 7.1 McKinsey -- QuantumBlack, AI by McKinsey

**Platform: QuantumBlack**

QuantumBlack is McKinsey's AI consulting arm, born from Formula 1 data analytics. Key offerings:

- **Hybrid Intelligence approach**: Combines AI/data technology with human strategic thinking and domain expertise
- **QuantumBlack Labs**: R&D center for AI tools, with technologists, designers, and product managers building cutting-edge assets
- **Open ecosystem of alliances**: Strategic partnerships with major AI/tech companies for generative AI implementations
- **Focus areas**: AI, Data Transformation, Noble Intelligence (social impact), Digital Twins

**Relevant to strategy alignment:**
- McKinsey uses proprietary AI tools internally for strategy analysis (not publicly available)
- They emphasize "real-world impact" over technology sophistication
- Their approach is to blend AI with domain expertise rather than pure automation
- Tools are typically bespoke for each client engagement

**Key insight for SDP:** McKinsey validates the "hybrid" approach -- AI augments human strategy work rather than replacing it. This aligns with the AI Architect's observed → proposed → reference progression.

### 7.2 BCG -- BCG X

**Platform: BCG X (formerly BCG Digital Ventures, GAMMA, and Platinion)**

BCG X is their tech build and design unit. Key offerings:
- AI strategy and implementation consulting
- Custom AI product development
- Data platform engineering

**Relevant tools (not publicly available as products):**
- BCG uses proprietary frameworks for strategy decomposition and alignment
- Focus on "bionic" companies that blend human and AI capabilities
- Internal tools for scenario planning and strategy simulation

### 7.3 Deloitte

**Platform: Deloitte AI & Data practice**

- **Deloitte AI Institute**: Research and thought leadership on enterprise AI
- **Omnia AI**: Deloitte's AI practice with industry-specific solutions
- **Strategy analytics**: Tools for strategic planning with data-driven insights

**Relevant offerings:**
- Document intelligence for regulatory compliance (similar techniques to strategy alignment)
- Knowledge graph solutions for enterprise information management
- NLP-based analysis of corporate documents for M&A due diligence

### 7.4 Accenture

**Platform: Accenture Data & AI**

- Enterprise-scale AI implementations
- Integration with major platforms (Microsoft, Google, AWS, SAP)
- Industry-specific AI solutions

### 7.5 Key Pattern Across All Consulting Firms

**None of them sell a standalone "strategy alignment checking tool."** Their approach is consistently:

1. **Consultant-driven**: AI tools augment consultants, not replace them
2. **Custom-built**: Solutions are tailored to each client's document corpus and strategy framework
3. **Integration-heavy**: Tools connect to the client's existing systems (Confluence, SharePoint, Jira, etc.)
4. **Human-in-the-loop**: Final alignment judgments are always made by humans

**Gap/opportunity:** There is no self-serve product that lets an engineering team check if their quarterly plans align with the company strategy. This is exactly the whitespace SDP AI Architect could fill.

---

## 8. Open-Source Projects & Repositories

### Key repositories for building strategy alignment analysis

### 8.1 Core RAG / Document Intelligence Frameworks

| Repository | Stars (approx) | Key Capability | Relevance |
|-----------|----------------|----------------|-----------|
| **run-llama/llama_index** | 40K+ | Document agent platform, 300+ integrations, hierarchical indexing, parent-child retrieval | Best for building the document ingestion and RAG pipeline |
| **langchain-ai/langchain** | 100K+ | Agent engineering platform, LangGraph for orchestration | Good for multi-step alignment analysis workflows |
| **deepset-ai/haystack** | 19K+ | Production AI pipelines, modular components, MCP server support | Used by enterprises (Airbus, Netflix); strong production story |
| **microsoft/graphrag** | 20K+ | Graph-based RAG for comprehensive reasoning over corpora | Best for building strategy knowledge graphs |

### 8.2 Knowledge Graph Tools

| Repository | Key Capability | Relevance |
|-----------|----------------|-----------|
| **neo4j/neo4j-graphrag-python** | Official Neo4j GraphRAG Python package; KG construction, vector search, hybrid retrieval | First-party KG-RAG integration with multiple LLM providers |
| **microsoft/graphrag** | LLM-powered entity/relationship extraction from unstructured text | Alternative to Neo4j for lighter-weight graph construction |

### 8.3 Integration Connectors

| Repository | Key Capability | Relevance |
|-----------|----------------|-----------|
| **atlassian-api/atlassian-python-api** | Python REST API wrapper for Jira, Confluence, Bitbucket, Service Desk | Primary integration path for Atlassian strategy documents |
| **git** (built-in) | Repository file access, markdown parsing | For git-backed strategy docs |

### 8.4 Embedding & Similarity

| Resource | Key Capability | Relevance |
|----------|----------------|-----------|
| **MTEB Leaderboard** (HuggingFace) | Benchmark ranking of embedding models across tasks | For selecting the best embedding model for strategy document similarity |
| **sentence-transformers** | Framework for computing sentence/text embeddings | The standard library for semantic similarity computation |
| **Qdrant, Weaviate, ChromaDB** | Vector databases for storing and querying embeddings | For storing document chunk embeddings with metadata filtering |

### 8.5 Relevant Research Papers (arxiv)

Key search terms on arxiv that yield relevant results:
- "strategy document analysis LLM"
- "organizational alignment NLP"
- "knowledge graph enterprise documents"
- "semantic similarity strategy"
- "automated requirements traceability NLP"
- "RAG enterprise document analysis"

The arxiv search returned 120K+ results for strategy-related queries. Key areas:
- Requirements traceability using NLP (adaptable to strategy traceability)
- Document similarity using transformer embeddings
- Knowledge graph construction from unstructured text
- Hierarchical document classification

---

## 9. Key Takeaways for SDP AI Architect

### What this research means for the AI Architect strategy alignment feature

### 9.1 Architecture Recommendation

Based on the research, the recommended architecture for strategy alignment analysis in SDP:

```
[Connector Layer]           -- Abstract interface to Confluence, Jira, YouTrack, Git, Local FS
        |
        v
[Document Ingestion]        -- Parse, chunk, classify hierarchy level, extract metadata
        |
        v
[Concept Extraction]        -- LLM structured output: objectives, initiatives, KPIs, risks, dependencies
        |
        v
[Graph Construction]        -- Build lightweight strategy graph (not necessarily Neo4j)
        |                      In-memory or SQLite for SDP's scope
        v
[Alignment Analysis]        -- Three-phase:
        |                      1. Semantic similarity triage (fast, cheap)
        |                      2. LLM reasoning assessment (slow, nuanced)
        |                      3. Graph traversal verification (structural)
        v
[Report Generation]         -- Structured alignment report with:
                               - Gap list (objectives with no decomposition)
                               - Orphan list (initiatives with no parent)
                               - Conflict list (contradictory priorities)
                               - Similarity scores per document pair
                               - Evidence citations
```

### 9.2 Implementation Priorities

1. **Start with local files** (markdown, PDF) -- no connector complexity
2. **Add Confluence connector** -- most common strategy document store
3. **Add Jira connector** -- for initiative/task level alignment
4. **Add Git connector** -- for ADR and docs-in-repo alignment

### 9.3 Key Design Decisions

| Decision | Recommendation | Rationale |
|----------|---------------|-----------|
| Graph database | **Lightweight in-process** (not Neo4j) | SDP is a CLI tool; no external DB dependency. Use adjacency lists or SQLite. |
| Embedding model | **OpenAI text-embedding-3-large** or **BGE-M3** | High quality; BGE-M3 is open source for air-gapped environments |
| RAG framework | **None initially** | Use direct LLM calls with CodebaseProfile pattern (same as existing Architect design). Add RAG when corpus exceeds context window. |
| Hierarchy detection | **Metadata + LLM fallback** | Use Confluence page tree / Jira issue type first, LLM classification when metadata unavailable |
| Alignment scoring | **Hybrid: similarity + LLM + graph** | Each approach catches different misalignment types |
| Output format | **JSON report + Markdown summary + Mermaid traceability diagram** | Consistent with existing AI Architect artifact patterns |

### 9.4 What NOT to Build

- Do not build a full RAG pipeline initially -- the CodebaseProfile compression pattern (from existing AI Architect design) handles most strategy document sets
- Do not require Neo4j or any external database
- Do not build a UI -- SDP is CLI-first; output artifacts that other tools can render
- Do not try to replace OKR tools (Lattice, Ally, Gtmhub) -- focus on document analysis
- Do not automate final alignment judgments -- flag and recommend, let humans decide

### 9.5 Competitive Whitespace

No existing tool provides:
1. **Self-serve strategy alignment checking** for engineering teams
2. **Multi-system integration** (Confluence + Jira + Git + local files) in a single analysis
3. **Document hierarchy awareness** with automatic level classification
4. **CLI-first** approach that fits into developer workflows
5. **Open source** strategy alignment analysis

This is a defensible niche for SDP AI Architect.

---

## Sources

- [Anthropic: Building Effective AI Agents](https://www.anthropic.com/research/building-effective-agents) -- Agent patterns (prompt chaining, routing, evaluator-optimizer, orchestrator-workers)
- [LlamaIndex GitHub](https://github.com/run-llama/llama_index) -- Document agent platform with LlamaParse, Extract, Index, Agents
- [LangChain GitHub](https://github.com/langchain-ai/langchain) -- Agent engineering platform with LangGraph, LangSmith
- [Haystack GitHub](https://github.com/deepset-ai/haystack) -- Production AI orchestration framework
- [Microsoft GraphRAG GitHub](https://github.com/microsoft/graphrag) -- Graph-based RAG system
- [Neo4j GraphRAG Python GitHub](https://github.com/neo4j/neo4j-graphrag-python) -- Official Neo4j GraphRAG with KG construction, vector search, hybrid retrieval
- [Atlassian Python API GitHub](https://github.com/atlassian-api/atlassian-python-api) -- REST API wrapper for Jira, Confluence, Bitbucket
- [Confluence Cloud REST API v2](https://developer.atlassian.com/cloud/confluence/rest/v2/intro/) -- Cursor-based pagination, authentication docs
- [MTEB Leaderboard](https://huggingface.co/spaces/mteb/leaderboard) -- Embedding model benchmark
- [McKinsey QuantumBlack](https://www.mckinsey.com/capabilities/quantumblack/how-we-help-clients) -- AI consulting arm approach
- [YouTrack](https://www.jetbrains.com/youtrack/) -- JetBrains project management with knowledge base, self-hosted option

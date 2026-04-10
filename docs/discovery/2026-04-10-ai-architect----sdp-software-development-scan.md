# Discovery Scan

**Whitespace:** The primary gap is a truly language-agnostic tool that can automatically detect common architectural patterns (e.g., microservices, layered architecture, event-driven) and generate C4 diagrams directly from code analysis, especially in polyglot brownfield projects, without extensive manual configuration or domain-specific language (DSL) definitions. There's also a lack of tools that explicitly focus on generating and validating integration/data contracts across different services and languages based on code analysis.

## Landscape

| Tool | Disposition | Coverage | Flagged |
|---|---|---|---|
| Understand.io | ADOPT | 0.03 | ⚠️ no_primary_source |
| Structure101 | ADOPT | 0.03 | ⚠️ no_primary_source |
| ArchUnit | EXTRACT | 0.03 | ⚠️ no_primary_source |
| PlantUML | INSPIRE | 0.03 | ⚠️ no_primary_source |
| SonarQube | MONITOR | 0.03 | ⚠️ no_primary_source |
| CodeScene | ADOPT | 0.03 | ⚠️ no_primary_source |
| Lattix | ADOPT | 0.03 | ⚠️ no_primary_source |

```json
{
  "items": [
    {
      "name": "Understand.io",
      "disposition": "ADOPT",
      "disposition_confidence": 0.7,
      "stars": 4,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 2,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.025,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": true
      },
      "key_strength": "Deep code analysis for multiple languages, providing metrics and visualizations.",
      "key_gap": "Limited explicit support for architectural pattern detection and automated C4 diagram generation.",
      "covers_phases": [
        "scan",
        "validate"
      ]
    },
    {
      "name": "Structure101",
      "disposition": "ADOPT",
      "disposition_confidence": 0.7,
      "stars": 4,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 2,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.025,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": true
      },
      "key_strength": "Focus on architectural conformance, dependency analysis, and drift detection.",
      "key_gap": "May require significant manual configuration for polyglot environments and lacks automated C4 generation.",
      "covers_phases": [
        "scan",
        "validate"
      ]
    },
    {
      "name": "ArchUnit",
      "disposition": "EXTRACT",
      "disposition_confidence": 0.6,
      "stars": 3,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 2,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.025,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": true
      },
      "key_strength": "Code-based architectural testing for Java, allowing definition of architectural rules.",
      "key_gap": "Primarily Java-specific, not suitable for polyglot environments without significant custom work.",
      "covers_phases": [
        "validate"
      ]
    },
    {
      "name": "PlantUML",
      "disposition": "INSPIRE",
      "disposition_confidence": 0.6,
      "stars": 3,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 2,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.025,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": false
      },
      "key_strength": "Text-based diagram generation, including C4 model, allowing for version control.",
      "key_gap": "Requires manual input for diagram generation; no automated code analysis or pattern detection.",
      "covers_phases": [
        "frame"
      ]
    },
    {
      "name": "SonarQube",
      "disposition": "MONITOR",
      "disposition_confidence": 0.6,
      "stars": 3,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 2,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.025,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": false
      },
      "key_strength": "Broad language support for code quality, security, and technical debt analysis.",
      "key_gap": "Focuses on code quality and security, not deep architectural understanding or C4 diagram generation.",
      "covers_phases": [
        "scan"
      ]
    },
    {
      "name": "CodeScene",
      "disposition": "ADOPT",
      "disposition_confidence": 0.7,
      "stars": 4,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 2,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.025,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": true
      },
      "key_strength": "Evolutionary architecture analysis, identifying hotspots and architectural decay.",
      "key_gap": "Less focused on explicit architectural pattern detection or automated C4 diagram generation.",
      "covers_phases": [
        "scan",
        "hypothesize"
      ]
    },
    {
      "name": "Lattix",
      "disposition": "ADOPT",
      "disposition_confidence": 0.7,
      "stars": 4,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 2,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.025,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": true
      },
      "key_strength": "Dependency Structure Matrix (DSM) for architectural analysis and conformance.",
      "key_gap": "Can be complex to set up and maintain, and its visualization is primarily DSM-based, not C4.",
      "covers_phases": [
        "scan",
        "validate"
      ]
    }
  ],
  "whitespace": "The primary gap is a truly language-agnostic tool that can automatically detect common architectural patterns (e.g., microservices, layered architecture, event-driven) and generate C4 diagrams directly from code analysis, especially in polyglot brownfield projects, without extensive manual configuration or domain-specific language (DSL) definitions. There's also a lack of tools that explicitly focus on generating and validating integration/data contracts across different services and languages based on code analysis.",
  "recommended_stack": [
    "Understand.io",
    "Structure101",
    "CodeScene",
    "PlantUML (for manual C4 refinement)"
  ],
  "cost_usd": 0.0033865
}
```

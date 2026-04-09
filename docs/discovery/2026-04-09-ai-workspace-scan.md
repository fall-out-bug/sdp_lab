# Discovery Scan

**Whitespace:** No tool fully addresses the complete workflow of automatically extracting insights from AI workspace dialogues, building a searchable knowledge base specifically for consultant projects, finding similar historical cases across projects, and providing intelligent conversation compression/rehydration features within AI workspaces.

## Landscape

| Tool | Disposition | Coverage | Flagged |
|---|---|---|---|
| Mem.ai | ADOPT | 0.04 | ⚠️ no_primary_source |
| Notion AI | EXTRACT | 0.04 | ⚠️ no_primary_source |
| Glean | INSPIRE | 0.04 | ⚠️ no_primary_source |
| Rewind.ai | INSPIRE | 0.04 | ⚠️ no_primary_source |
| Fireflies.ai | EXTRACT | 0.04 | ⚠️ no_primary_source |
| Obsidian | INSPIRE | 0.04 | ⚠️ no_primary_source |
| ChatGPT with Memory | MONITOR | 0.04 | ⚠️ no_primary_source |
| Microsoft Copilot for Microsoft 365 | MONITOR | 0.04 | ⚠️ no_primary_source |

```json
{
  "items": [
    {
      "name": "Mem.ai",
      "disposition": "ADOPT",
      "disposition_confidence": 0.6,
      "stars": 4,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 3,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.0375,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": true
      },
      "key_strength": "Automatic note-taking and knowledge organization from conversations and documents with AI-powered search and connections",
      "key_gap": "Limited conversation-specific compression/rehydration features; more general knowledge management",
      "covers_phases": [
        "frame",
        "scan",
        "validate"
      ]
    },
    {
      "name": "Notion AI",
      "disposition": "EXTRACT",
      "disposition_confidence": 0.7,
      "stars": 3,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 3,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.0375,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": true
      },
      "key_strength": "Integrated AI within a flexible workspace for summarizing, extracting insights, and organizing project knowledge",
      "key_gap": "Not specifically designed for dialogue analysis or conversation history management",
      "covers_phases": [
        "frame",
        "scan"
      ]
    },
    {
      "name": "Glean",
      "disposition": "INSPIRE",
      "disposition_confidence": 0.5,
      "stars": 4,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 3,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.0375,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": false
      },
      "key_strength": "Enterprise search across multiple data sources with AI to find relevant information and past conversations",
      "key_gap": "Focus on search rather than dialogue insight extraction or context compression",
      "covers_phases": [
        "scan",
        "validate"
      ]
    },
    {
      "name": "Rewind.ai",
      "disposition": "INSPIRE",
      "disposition_confidence": 0.6,
      "stars": 3,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 3,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.0375,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": false
      },
      "key_strength": "Records and indexes everything on your computer, enabling search through past conversations and meetings",
      "key_gap": "Privacy concerns; not specifically designed for project knowledge distillation",
      "covers_phases": [
        "scan",
        "validate"
      ]
    },
    {
      "name": "Fireflies.ai",
      "disposition": "EXTRACT",
      "disposition_confidence": 0.7,
      "stars": 3,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 3,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.0375,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": true
      },
      "key_strength": "Automated meeting transcription, note-taking, and insight extraction with search capabilities",
      "key_gap": "Meeting-focused rather than broader project dialogue and workspace context management",
      "covers_phases": [
        "frame",
        "scan"
      ]
    },
    {
      "name": "Obsidian",
      "disposition": "INSPIRE",
      "disposition_confidence": 0.8,
      "stars": 4,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 3,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.0375,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": false
      },
      "key_strength": "Local-first knowledge base with powerful linking and graph view for connecting ideas across notes",
      "key_gap": "Manual note-taking required; no automatic dialogue insight extraction",
      "covers_phases": [
        "frame",
        "scan"
      ]
    },
    {
      "name": "ChatGPT with Memory",
      "disposition": "MONITOR",
      "disposition_confidence": 0.7,
      "stars": 3,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 3,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.0375,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": false
      },
      "key_strength": "AI assistant that can remember past conversations and reference them in current discussions",
      "key_gap": "Limited to ChatGPT ecosystem; not designed for cross-project knowledge management",
      "covers_phases": [
        "frame",
        "scan"
      ]
    },
    {
      "name": "Microsoft Copilot for Microsoft 365",
      "disposition": "MONITOR",
      "disposition_confidence": 0.6,
      "stars": 3,
      "source_count": 1,
      "primary_source_read": false,
      "architecture_reviewed": false,
      "desc_sentences": 3,
      "multi_source": false,
      "age_months": 0,
      "coverage_score": 0.0375,
      "depth_flag": {
        "flagged": true,
        "reason": "no_primary_source",
        "recommended_action": "deep_dive",
        "blocking": false
      },
      "key_strength": "AI integration across Microsoft ecosystem with access to organizational data and past conversations",
      "key_gap": "Enterprise-focused with limited customization for specific consultant workflows",
      "covers_phases": [
        "scan",
        "validate"
      ]
    }
  ],
  "whitespace": "No tool fully addresses the complete workflow of automatically extracting insights from AI workspace dialogues, building a searchable knowledge base specifically for consultant projects, finding similar historical cases across projects, and providing intelligent conversation compression/rehydration features within AI workspaces.",
  "recommended_stack": [
    "Mem.ai for knowledge organization",
    "Fireflies.ai for meeting insight extraction",
    "Obsidian for connecting insights across projects"
  ],
  "cost_usd": 0.000621955
}
```

# Agent Skills Specification

Status: reference  
Source: `specs/agent-skills.yaml`

## Overview

Role-specific skills are loaded by `SkillRegistry` and influence agent behavior (prompts, tool selection, reasoning style).

## Roles and Skills

| Role | Skills |
|------|--------|
| analyst | requirement-decomposition, risk-analysis, dependency-mapping |
| coder | code-generation, test-writing, refactoring, boundary-compliance |
| reviewer | adversarial-review, consensus-scoring, feedback-structuring |
| retro | telemetry-analysis, pattern-detection, improvement-proposal |
| orchestrator | scheduling, lifecycle-management, dispatch |

## Defaults

When `specs/agent-skills.yaml` is absent or a role is not listed, defaults are used.

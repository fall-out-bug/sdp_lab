# Persona Registry Spec

Status: baseline v1
Scope: evaluator swarm persona configuration

## 1. Format

YAML file at `specs/persona-registry.yaml`:

```yaml
personas:
  - id: systems-architect
    decision_lens: "..."
    primary_question: "..."
    required_evidence: [...]
    escalation_target: product-strategist
    model: glm-5
```

## 2. Fields

| Field | Required | Description |
|-------|----------|-------------|
| id | yes | Unique persona identifier; maps to Agent CR name |
| decision_lens | yes | Role-specific evaluation lens |
| primary_question | yes | Key question for this persona |
| required_evidence | no | Evidence IDs this persona needs |
| escalation_target | no | Persona ID to escalate to on conflict |
| model | no | Model for this persona (default: glm-5) |

## 3. Agent CR mapping

Each persona maps to a kubeopencode Agent CR:

- `metadata.name`: `sdp-<persona-id>` (e.g. sdp-systems-architect)
- `spec.model`: from persona registry
- `contexts[0].text`: role prompt including persona_id

## 4. Extensibility

To add a new persona:

1. Append entry to `specs/persona-registry.yaml`
2. Add Agent CR to `deploy/k8s/kubeopencode/agents.yaml`
3. No code changes in evaluator core

## 5. References

- [specs/persona-registry.yaml](../specs/persona-registry.yaml)
- [deploy/k8s/kubeopencode/agents.yaml](../deploy/k8s/kubeopencode/agents.yaml)

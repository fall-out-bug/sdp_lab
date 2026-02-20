# Real Feature-to-PR Runbook (Private)

This runbook executes the full operator path from task claim to live PR.

## Preconditions

- remote origin is configured and has a default branch
- `gh auth status` is valid
- target task exists in Beads with `autonomy` and `strict-evidence` labels

## Steps

1. Claim task (autonomous):

```bash
go run ./cmd/autonomy-worker
```

2. Validate transition to review:

```bash
go run ./cmd/beads-fsm --issue <issue-id> --to review --apply
```

3. Prepublish strict-evidence gate:

```bash
go run ./cmd/pr-gate --issue <issue-id> --prepublish
```

4. Push feature branch:

```bash
git push -u origin <feature-branch>
```

5. Publish PR and write `trace.pr_url`:

```bash
go run ./cmd/pr-publish --issue <issue-id> --title "..." --head <feature-branch> --base master --body-file <body.md>
```

6. Validate publish gate:

```bash
go run ./cmd/pr-gate --issue <issue-id>
```

7. Complete flow and close task:

```bash
go run ./cmd/beads-fsm --issue <issue-id> --to verified --apply
go run ./cmd/beads-fsm --issue <issue-id> --to done --apply
```

## Policy notes

- merge is always manual (human gate)
- model allowlist remains `glm-5`, `glm-4.7`

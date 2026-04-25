# Skill Template Marker Convention — F131

**Version**: 1.0.0  
**Date**: 2026-04-25  
**Purpose**: Define marker convention for stack-specific sections in skill templates

## Overview

Skills in `.agents/skills/` sometimes need language/stack-specific content (e.g., `go test` vs `pytest`, `npm run build` vs `cargo build`). This marker convention allows downstream tooling (F131-02 augmenter) to inject stack-specific blocks without clobbering hand-written content.

## Marker Syntax

### Basic Marker

```html
<!-- STACK_SPECIFIC:BEGIN section="<section-name>" stack="<stack-name>" -->
<stack-specific content here>
<!-- STACK_SPECIFIC:END -->
```

### Parameters

- **section** (required): The type of content being marked
  - Supported values: `test`, `build`, `lint`, `debug`, `quality-gate`, `coverage`
- **stack** (required): The technology stack this content applies to
  - Supported values: `go`, `python`, `typescript`, `rust`, `java`, `javascript`, `csharp`, `ruby`, `php`, `swift`, `kotlin`, `dart`, `other`

### Complete Example

```html
<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
## Testing Commands

Run all tests:
```bash
go test ./... -v
```

Run with coverage:
```bash
go test -coverprofile=/tmp/coverage.out ./...
go tool cover -html=/tmp/coverage.out
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="python" -->
## Testing Commands

Run all tests:
```bash
pytest -v
```

Run with coverage:
```bash
pytest --cov=. --cov-report=html
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="typescript" -->
## Testing Commands

Run all tests:
```bash
npm test
```

Run with coverage:
```bash
npm run test:cov
```
<!-- STACK_SPECIFIC:END -->
```

## Supported Sections

| Section | Description | Example Use Cases |
|---------|-------------|-------------------|
| **test** | Test execution commands | `go test`, `pytest`, `npm test`, `cargo test` |
| **build** | Build compilation commands | `go build`, `npm run build`, `cargo build`, `mvn compile` |
| **lint** | Linting/static analysis | `go vet`, `flake8`, `eslint`, `cargo clippy` |
| **debug** | Debugging commands | `dlv`, `pdb`, `node --inspect`, `gdb` |
| **quality-gate** | Quality gate commands | Combined test+build+lint checks |
| **coverage** | Coverage analysis | `go tool cover`, `pytest-cov`, `nyc` |

## Supported Stacks

| Stack | Description | Primary Tools |
|-------|-------------|---------------|
| **go** | Go/Golang | go test, go build, go vet |
| **python** | Python | pytest, pip, flake8 |
| **typescript** | TypeScript/Node.js | npm, npx, tsc |
| **javascript** | JavaScript/Node.js | npm, node |
| **rust** | Rust | cargo, rustc |
| **java** | Java | mvn, gradle, javac |
| **csharp** | C#/.NET | dotnet |
| **ruby** | Ruby | ruby, bundle, rake |
| **php** | PHP | composer, phpunit |
| **swift** | Swift | swift, xcodebuild |
| **kotlin** | Kotlin | gradle, kotlinc |
| **dart** | Dart | dart, flutter |
| **other** | Catch-all for other stacks | (varies) |

## Which Skills Should Use Markers

Based on the [F131-01 audit](./skill-stack-audit-f131.md), the following 14 skills benefit from stack-specific sections:

1. **test-coverage.md** - Coverage analysis per stack
2. **test-writer.md** - Test generation per stack
3. **smoke-test.md** - Smoke test commands per stack
4. **build.md** - Build/test commands per stack ⭐ POC target
5. **fix.md** - Debugging commands per stack
6. **feature-delivery.md** - Quality gates per stack
7. **bug-fix.md** - Test commands per stack
8. **git-worktree.md** - Dependency detection per stack
9. **deploy.md** - Deploy commands per stack
10. **operate.md** - DevOps commands per stack
11. **prototype.md** - Prototyping commands per stack
12. **oneshot.md** - Quick commands per stack
13. **hotfix.md** - Hotfix commands per stack
14. **bugfix.md** - Legacy bugfix commands per stack

## Idempotency Contract

**Re-running marker injection MUST produce identical output.**

### Rules for Augmenter (F131-02)

1. **Only replace marker-bounded blocks**: Never modify content outside `<!-- STACK_SPECIFIC:BEGIN -->` ... `<!-- STACK_SPECIFIC:END -->`
2. **Exact marker matching**: Match on full marker string including parameters
3. **Preserve surrounding content**: Content before/after markers remains unchanged
4. **Append-only for new stacks**: If a stack marker doesn't exist, append it after existing markers
5. **Never duplicate**: If a stack marker exists, replace its content entirely (don't create duplicates)

### Example: Idempotent Injection

**Initial state:**
```markdown
<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->
```

**After first injection (add Python):**
```markdown
<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="python" -->
pytest -v
<!-- STACK_SPECIFIC:END -->
```

**After second injection (idempotent - no change):**
```markdown
<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="python" -->
pytest -v
<!-- STACK_SPECIFIC:END -->
```

**After third injection (update Go content):**
```markdown
<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./... -v
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="python" -->
pytest -v
<!-- STACK_SPECIFIC:END -->
```

## Marker Placement Guidelines

### Where to Place Markers

1. **After general content**: Place stack-specific sections after general skill description
2. **Before "References"**: Place before reference sections if they exist
3. **Group by section**: Keep all `section="test"` markers together, then all `section="build"` markers
4. **Logical flow**: Test → Build → Lint → Debug (typically follows workflow order)

### Example Structure

```markdown
# Skill Name

## Purpose
<general description>

## When to Use
<general usage>

## MUST DO
<general requirements>

<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
...
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="python" -->
...
<!-- STACK_SPECIFIC:END -->

## References
<general references>
```

## Implementation Notes

### For Skill Authors

1. Start with general content (applies to all stacks)
2. Add marker blocks for stack-specific sections
3. Keep general content minimal - move stack-specific details into markers
4. Always include at least 2 stacks in a section (avoid single-stack markers)

### For Augmenter Authors (F131-02)

1. Parse markdown file for existing markers
2. For each `(section, stack)` pair:
   - If marker exists: replace content between BEGIN/END
   - If marker doesn't exist: append after existing markers in same section
3. Never modify content outside markers
4. Validate that all markers have matching BEGIN/END pairs
5. Fail gracefully if marker syntax is invalid (don't clobber file)

## Testing

### Test Cases for Idempotency

1. **Fresh injection**: Inject markers into file without any markers
2. **Update injection**: Inject markers into file with existing markers (should update)
3. **Idempotent injection**: Inject identical markers twice (should produce same output)
4. **Multi-section injection**: Inject multiple sections (test, build, lint)
5. **Multi-stack injection**: Inject multiple stacks (go, python, typescript)
6. **Partial injection**: Inject only new stacks, preserve existing stacks
7. **Content preservation**: Ensure content outside markers is unchanged

### Validation Script

```bash
# Validate marker syntax in a skill file
validate_markers() {
    local file="$1"
    
    # Count BEGIN/END markers (must match)
    local begin_count=$(grep -c '<!-- STACK_SPECIFIC:BEGIN' "$file")
    local end_count=$(grep -c '<!-- STACK_SPECIFIC:END -->' "$file")
    
    if [ "$begin_count" -ne "$end_count" ]; then
        echo "ERROR: Mismatched markers in $file"
        return 1
    fi
    
    # Validate marker syntax
    grep '<!-- STACK_SPECIFIC:BEGIN' "$file" | while read -r line; do
        if ! echo "$line" | grep -qE 'section="[^"]+"'; then
            echo "ERROR: Missing section parameter in: $line"
            return 1
        fi
        if ! echo "$line" | grep -qE 'stack="[^"]+"'; then
            echo "ERROR: Missing stack parameter in: $line"
            return 1
        fi
    done
    
    echo "OK: $file has $begin_count valid marker pairs"
    return 0
}

# Validate all skills
for skill in .agents/skills/*.md; do
    validate_markers "$skill"
done
```

## Future Enhancements

1. **Stack detection**: Auto-detect stack from project files (go.mod, package.json, etc.)
2. **Conditional rendering**: Hide irrelevant stacks based on detected project stack
3. **Stack aliases**: Allow `stack="node"` to render TypeScript content
4. **Nested sections**: Support nested markers for complex scenarios
5. **Marker inheritance**: Allow base templates to be extended by stack-specific templates

## References

- [F131-01 Skills Audit](./skill-stack-audit-f131.md) - Complete classification of 43 skills
- [F131-02 Augmenter Implementation](../plans/2026-04-25-f131-02-augmenter-design.md) - Downstream tooling that consumes these markers
- [Markdown Specification](https://spec.commonmark.org/) - Standard markdown syntax

---
name: nim-dev
description: Nim development, testing, and quality assurance. Use for systems programming, game dev, web backends, CLI tools, and Python interop.
---

# Nim Development

## Top 10 Patterns

1. ** testament** — built-in test runner (`nim r tests/test.nim`)
2. ** unittest / check** — assertions
3. **property-based testing (rapid)** — via `quickcheck` or `rapid`
4. **nimpretty** — formatting
5. **nimble** — package manager and build tool
6. **memtrace** — memory profiling
7. **c2nim** — C header wrapping
8. **nim doc** — documentation
9. **ARC/ORC** — deterministic memory management
10. ** pragmas `{.gcsafe.}` `{.thread.}`** — concurrency safety

## Quality Gates

```bash
nimble test
nim c -r tests/all
nim pretty --check
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `testament` | Test runner |
| `unittest` | Framework |
| `nimble` | Package manager |
| `nimpretty` | Format |
| `nim doc` | Docs |
| `gc:arc/orc` | Memory mgmt |

---
name: ocaml-dev
description: OCaml development, testing, and quality assurance. Use for compilers, formal verification, financial systems, and high-assurance software.
---

# OCaml Development

## Top 10 Patterns

1. **dune** — modern build system
2. **Alcotest** — composable testing framework
3. **QCheck** — property-based testing
4. **bisect_ppx** — code coverage
5. **ocamlformat** — formatting
6. **merlin / ocaml-lsp** — editor support
7. **opam** — package management
8. **ppx** — metaprogramming / code generation
9. **GADT / phantom types** — type-safe APIs
10. **Lwt / Async** — concurrent programming

## Quality Gates

```bash
dune runtest
dune build @check
ocamlformat --check
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `dune runtest` | Testing |
| `alcotest` | Framework |
| `qcheck` | Property testing |
| `bisect_ppx` | Coverage |
| `ocamlformat` | Format |
| `merlin` | IDE support |
| `opam` | Packages |

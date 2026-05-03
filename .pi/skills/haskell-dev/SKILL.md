---
name: haskell-dev
description: Haskell development, testing, and quality assurance. Use for functional programming, compilers, formal methods, financial systems, and high-assurance software.
---

# Haskell Development

## Top 10 Patterns

1. **Stack / Cabal** — declarative dependency management
2. **Hspec / tasty** — expressive test frameworks
3. **QuickCheck / Hedgehog** — property-based testing
4. **Hlint** — linting and refactoring suggestions
5. **Ormolu / Fourmolu** — formatting
6. **GHC extensions** — `{-# LANGUAGE ... #-}` with care
7. **STM (Software Transactional Memory)** — composable concurrency
8. **lens / optics** — functional data access
9. **criterion** — benchmarking
10. **doctest** — test examples in documentation

## Quality Gates

```bash
stack test --coverage
hlint .
ormolu --mode check $(find . -name '*.hs')
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `stack test` / `cabal test` | Testing |
| `hspec` / `tasty` | Frameworks |
| `QuickCheck` | Property testing |
| `hlint` | Lint |
| `ormolu` / `fourmolu` | Format |
| `criterion` | Benchmark |
| `weeder` | Dead code |
| `stan` | Static analysis |

---
name: julia-dev
description: Julia development, testing, and quality assurance. Use for scientific computing, data science, ML, differential equations, and high-performance numerical work.
---

# Julia Development

## Top 10 Patterns

1. **Pkg.test** — built-in test runner
2. **Test.jl** — standard library testing
3. **BenchmarkTools** — `@btime`, `@benchmark`
4. **Revise.jl** — hot code reloading in REPL
5. **Documenter.jl** — documentation generation
6. **JuliaFormatter** — code formatting
7. **JET.jl** — static analysis
8. **Aqua.jl** — package quality checks
9. ** multiple dispatch** — design core functions around it
10. **PackageCompiler** — sysimage creation for fast startup

## Quality Gates

```julia
using Pkg; Pkg.test()
using JuliaFormatter; format(".", check=true)
using JET; report_package("MyPackage")
using Aqua; test_all(MyPackage)
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `Pkg.test` | Testing |
| `Test` | Framework |
| `BenchmarkTools` | Benchmarks |
| `JET` | Static analysis |
| `Aqua` | Quality checks |
| `JuliaFormatter` | Format |
| `Documenter` | Docs |
| `Coverage.jl` | Coverage |

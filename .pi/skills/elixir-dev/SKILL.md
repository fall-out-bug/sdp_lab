---
name: elixir-dev
description: Elixir / Erlang development, testing, and quality assurance. Use for distributed systems, real-time apps, fault-tolerant services, and telecom-grade systems.
---

# Elixir Development

## Top 10 Patterns

1. **ExUnit** — built-in test framework with async support
2. **Phoenix / LiveView** — web and real-time UI
3. **GenServer / GenStage / Broadway** — OTP behaviours
4. **Ecto** — database layer with changesets
5. **Credo** — static analysis and style
6. **Dialyzer** — gradual type checking
7. **ExDoc** — documentation generation
8. **Property-based testing (StreamData)** — `check all` syntax
9. **Supervision trees** — fault tolerance by design
10. **Mix releases** — production deployment

## Quality Gates

```bash
mix test --cover
mix format --check-formatted
mix credo --strict
mix dialyzer
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `mix test` | Testing |
| `ex_unit` | Framework |
| `stream_data` | Property testing |
| `credo` | Static analysis |
| `dialyzer` | Type check |
| `excoveralls` | Coverage |
| `benchee` | Benchmark |
| `sobelow` | Security |

---
name: csharp-dev
description: C# / .NET development, testing, and quality assurance. Use for web APIs, desktop, games (Unity), cloud, and enterprise apps.
---

# C# / .NET Development

## Top 10 Patterns

1. **xUnit + FluentAssertions** — modern test stack
2. **NSubstitute / Moq** — mocking (prefer NSubstitute)
3. **WebApplicationFactory** — integration testing ASP.NET Core
4. **Testcontainers .NET** — Docker in tests
5. **BenchmarkDotNet** — performance benchmarking
6. **Verify / Snapshooter** — snapshot approval testing
7. **Stryker.NET** — mutation testing
8. **coverlet + ReportGenerator** — code coverage
9. **dotnet format + StyleCop** — consistent style
10. **NSwag / Swagger** — API contract validation

## Quality Gates

```bash
dotnet format --verify-no-changes
dotnet build --no-restore
dotnet test --no-build --verbosity normal
dotnet stryker  # mutation testing
```

## Key Tools

| Tool | Purpose |
|------|---------|
| xUnit / NUnit / MSTest | Test frameworks |
| FluentAssertions | Readable assertions |
| NSubstitute | Mocking |
| WebApplicationFactory | Integration tests |
| Testcontainers | Docker integration |
| BenchmarkDotNet | Benchmarks |
| Stryker.NET | Mutation testing |
| coverlet | Coverage |

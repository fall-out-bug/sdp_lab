---
name: go-dev
description: Go (Golang) development, testing, and quality assurance. Use for microservices, CLI tools, systems programming, and cloud-native applications.
---

# Go Development

## Top 10 Patterns

1. **Table-Driven Tests** — canonical Go pattern for test cases
2. **Testify** — `assert`, `require`, `suite`, `mock` for ergonomics
3. **gomock / mockery** — interface mocking
4. **TestContainers-Go** — Docker-based integration tests
5. **Benchmark with benchstat** — statistical comparison of benchmarks
6. **Fuzzing** — native `go test -fuzz` since Go 1.18
7. **Race Detector** — `go test -race` always in CI
8. **pprof + trace** — CPU, memory, goroutine profiling
9. **gofumpt + golines + goimports** — stricter than gofmt
10. **wire / fx** — dependency injection for larger apps

## Quality Gates

```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go vet ./...
golangci-lint run ./...
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `go test` | Native testing |
| `testify` | Assertions, mocks, suites |
| `ginkgo` + `gomega` | BDD style |
| `stretchr/testify/suite` | Test suites |
| `gotestsum` | Prettier test output |
| `tparse` | Parse go test output |
| `benchstat` | Benchmark comparison |
| `go-wrk` / `k6` | Load testing |

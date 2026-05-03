---
name: kotlin-dev
description: Kotlin Multiplatform / Native development, testing, and quality assurance. Use for Android, server-side, iOS (via KMP), WebAssembly, and native binaries.
---

# Kotlin Development

## Top 10 Patterns

1. **Kotlin Multiplatform (KMP)** — shared code across JVM, Native, JS
2. **Kotlin/Native** — LLVM-based native binaries
3. **Ktor** — async server/client framework
4. ** kotlinx.coroutines + Flow** — structured concurrency
5. **Kotest** — expressive testing with property-based support
6. **MockK** — idiomatic mocking for Kotlin
7. **Detekt** — static analysis
8. **ktlint** — formatting
9. **kotlinx.serialization** — multiplatform serialization
10. **Gradle Kotlin DSL** — type-safe build scripts

## Quality Gates

```bash
./gradlew check koverXmlReport
detekt
git diff --name-only --diff-filter=ACM | grep '\.kt$' | xargs ktlint
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `kotest` | Testing |
| `mockk` | Mocking |
| `detekt` | Static analysis |
| `ktlint` | Formatting |
| `kover` | Coverage |
| `kotlinx-benchmark` | Benchmarks |
| `turbolinks` | KMP testing |

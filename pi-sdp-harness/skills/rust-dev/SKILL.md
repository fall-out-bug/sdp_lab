---
name: rust-dev
description: Rust development, testing, and quality assurance. Use for systems programming, WASM, CLI tools, embedded, and high-performance services.
---

# Rust Development

## Top 10 Patterns

1. **Cargo workspaces** — monorepo management
2. **proptest / quickcheck** — property-based testing
3. **rstest** — fixture-based testing
4. **criterion.rs** — statistical benchmarking
5. **miri** — undefined behavior detection
6. **cargo fuzz** — fuzzing with libFuzzer
7. **tokio-test / async-trait** — async testing
8. **mockall / wiremock** — mocking
9. **cargo-audit / cargo-deny** — security audit
10. **clippy -- -D warnings** — lint as error

## Quality Gates

```bash
cargo test --all-features
cargo clippy --all-targets --all-features -- -D warnings
cargo fmt --check
cargo audit
cargo tarpaulin --out Xml  # coverage
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `cargo test` | Native testing |
| `cargo nextest` | Faster test runner |
| `criterion` | Benchmarks |
| `proptest` | Property testing |
| `mockall` | Mocking |
| `insta` | Snapshot testing |
| `cargo-tarpaulin` | Code coverage |
| `miri` | UB detection |

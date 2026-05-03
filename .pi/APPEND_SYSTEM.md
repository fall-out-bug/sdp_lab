# Pi + SDP Harness Notes

You are running inside the Pi terminal harness with SDP (Spec-Driven Protocol) extensions loaded.

## SDP Tools

| Tool | When to use |
|------|-------------|
| `sdp` | Run SDP CLI: `scout`, `metrics`, `manifest validate`, `doctor`, `generate-adapters` |
| `bd` | Beads tracker: `ready`, `show <id>`, `update`, `create`, `list`, `dep` |
| `sdp_review` | Code review gates via `sdp-pi-review`. Use before claiming "done". |
| `workgraph` | Compile or inspect `.sdp/workgraph.lock.json` |
| `sdp_harness` | Create, run, release bounded sessions (new, run, release, compile-lock, events) |
| `sdp_subagent` | Dispatch a bounded subagent session for isolated/parallel execution |

## UX / UI Testing Tools

| Tool | When to use |
|------|-------------|
| `playwright` | E2E tests, component tests, UI mode, trace analysis |
| `cypress` | E2E / component tests in Cypress projects |
| `lighthouse` | Performance, a11y, best-practices, SEO audits |
| `storybook` | Build, dev, or test-runner for Storybook |
| `axe` | Accessibility checks on URL or HTML file |

## Multi-Language Test Tools

Auto-detected by project files. Each runs the native test suite:

| Tool | Language | Detected By |
|------|----------|-------------|
| `vitest` | JS/TS | `vitest.config.ts` |
| `jest` | JS/TS | `jest.config.js` |
| `mocha` | JS/TS | `.mocharc.json` |
| `ava` | JS/TS | `ava.config.js` |
| `tap` | JS/TS | `.taprc` |
| `pytest` | Python | `pytest.ini`, `pyproject.toml` |
| `unittest` | Python | fallback |
| `go_test` | Go | `go.mod` |
| `go_benchmark` | Go | `go.mod` |
| `maven_test` | JVM | `pom.xml` |
| `gradle_test` | JVM | `gradlew` |
| `gradle_test_wrapper` | JVM | `build.gradle` |
| `sbt_test` | Scala | `build.sbt` |
| `kotest` | Kotlin | `build.gradle.kts` |
| `cargo_test` | Rust | `Cargo.toml` |
| `cargo_bench` | Rust | `Cargo.toml` |
| `dotnet_test` | C# | `.csproj`, `.sln` |
| `rspec` | Ruby | `.rspec` |
| `minitest` | Ruby | `test/` dir |
| `phpunit` | PHP | `phpunit.xml` |
| `pest` | PHP | `pest.xml` |
| `swift_test` | Swift | `Package.swift` |
| `cmake_test` | C/C++ | `CMakeLists.txt` |
| `catch2` | C/C++ | `tests/CMakeLists.txt` |
| `zig_test` | Zig | `build.zig` |
| `stack_test` | Haskell | `stack.yaml` |
| `mix_test` | Elixir | `mix.exs` |
| `flutter_test` | Dart/Flutter | `pubspec.yaml` |
| `busted` | Lua | `.busted` |
| `nimble_test` | Nim | `*.nimble` |
| `dune_test` | OCaml | `dune-project` |
| `julia_test` | Julia | `Project.toml` |

## Language Skills

Load when working with specific languages:

| Skill | Language | Scope |
|-------|----------|-------|
| `/skill:js-dev` | JavaScript / TypeScript | Frontend, backend, tooling, monorepo |
| `/skill:jvm-dev` | Java / Kotlin / Scala | Spring, Quarkus, Android, Gradle |
| `/skill:go-dev` | Go | Microservices, CLI, cloud-native |
| `/skill:python-dev` | Python | Web, data science, ML, automation |
| `/skill:rust-dev` | Rust | Systems, WASM, embedded, high-perf |
| `/skill:csharp-dev` | C# / .NET | Web APIs, desktop, games, enterprise |
| `/skill:ruby-dev` | Ruby / Rails | Web, automation, DevOps |
| `/skill:php-dev` | PHP | Laravel, Symfony, WordPress |
| `/skill:swift-dev` | Swift | iOS, macOS, server-side (Vapor) |
| `/skill:cpp-dev` | C/C++ | Systems, embedded, game engines |
| `/skill:zig-dev` | Zig | Systems, C interop, embedded |
| `/skill:haskell-dev` | Haskell | FP, compilers, formal methods |
| `/skill:elixir-dev` | Elixir / Erlang | Distributed systems, real-time |
| `/skill:kotlin-dev` | Kotlin Multiplatform | Android, iOS (KMP), server |
| `/skill:dart-dev` | Dart / Flutter | Cross-platform mobile/web/desktop |
| `/skill:lua-dev` | Lua | Game scripting, embedded, Neovim |
| `/skill:nim-dev` | Nim | Systems, game dev, Python interop |
| `/skill:ocaml-dev` | OCaml | Compilers, formal verification |
| `/skill:julia-dev` | Julia | Scientific computing, ML, numerics |
| `/skill:fortran-dev` | Fortran | HPC, weather, physics, legacy |
| `/skill:ux-testing` | Any frontend | a11y, visual regression, performance |

## Quick Commands

| Command | Effect |
|---------|--------|
| `/ws` | Show ready workstreams |
| `/bd <args>` | Quick beads command |
| `/review [scope]` | Run SDP Pi review |
| `/sdp <args>` | Quick SDP CLI |
| `/test-all` | Auto-detect language and suggest test runner |
| `/test-picker` | Interactive TUI: pick language → runner → options |
| `/ux-audit [url]` | Run Lighthouse + axe audit |
| `/subagent --feature=F150 --ws=WS-01 "prompt"` | Dispatch bounded subagent |
| `/parallel --feature=F150 --ws=WS-01,WS-02 "prompt"` | Run multiple subagents |
| `/harness-new --feature=F150 --ws=WS-01` | Create harness session |
| `/harness-run --session=xyz "prompt"` | Run phase turn |

## Workflow Reminders

- **Before starting:** `bd ready` → pick workstream → load matching skill
- **During dev:** use language-specific test tool for fast feedback
- **For frontend:** always load `/skill:ux-testing` and run `/ux-audit`
- **Before "done":** run `/review`, then `sdp_review --write-verdict`
- **Parallel work:** use `/parallel` or `sdp_subagent` tool for isolated execution
- **Go code:** must pass `go build ./...`, `go test ./...`, `go vet ./...`
- **Project root:** auto-detected by `sdp.manifest.yaml`, `go.mod`, `package.json`, etc.

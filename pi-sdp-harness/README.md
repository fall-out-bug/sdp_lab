# @fall-out-bug/pi-sdp-harness

Pi harness for SDP (Spec-Driven Protocol) workspaces.

## Install

```bash
pi install git:github.com/fall-out-bug/sdp_lab
```

Or with npm (if published):
```bash
pi install npm:@fall-out-bug/pi-sdp-harness
```

## What You Get

### Extensions
- `sdp.ts` — SDP CLI, beads, review, workgraph tools
- `multilang-test.ts` — UX/UI testing (Playwright, Cypress, Lighthouse, Storybook, axe) + 23 language test runners

### Skills (11)
- `js-dev` — JS/TS development patterns
- `jvm-dev` — Java/Kotlin/Scala development
- `go-dev` — Go development
- `python-dev` — Python development
- `rust-dev` — Rust development
- `csharp-dev` — C# / .NET development
- `ruby-dev` — Ruby / Rails development
- `php-dev` — PHP development
- `swift-dev` — Swift development
- `cpp-dev` — C/C++ development
- `ux-testing` — UX/UI testing and accessibility

### Prompts (25+)
SDP workflow prompts: `/feature`, `/build`, `/ship`, `/review`, `/beads`, etc.

## Usage

After install, in any SDP project:
```bash
pi
```

Commands:
- `/ws` — Show ready workstreams
- `/review` — Run SDP code review
- `/test-all` — Auto-detect language and run tests
- `/ux-audit` — Run Lighthouse + axe
- `/skill:js-dev` — Load JS development skill

---
name: zig-dev
description: Zig development, testing, and quality assurance. Use for systems programming, C interoperability, embedded, game engines, and incremental migration from C/C++.
---

# Zig Development

## Top 10 Patterns

1. **comptime** — compile-time execution for generics, type generation
2. **zig build** — native build system, declarative build.zig
3. **zig test** — built-in test runner with `test "name" { ... }`
4. **zig fmt** — mandatory formatting
5. **zig-cache** — explicit cache management
6. **C interop** — `@cImport`, `@cInclude`, translate-c
7. **error unions** — `!T` for explicit error handling
8. **General Purpose Allocator** — `std.heap.GeneralPurposeAllocator`
9. **zig build-obj/lib/exe** — cross-compilation targets
10. **zig libc** — bundle libc for freestanding targets

## Quality Gates

```bash
zig build test
zig fmt --check .
zig build -Dtarget=x86_64-linux-gnu
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `zig test` | Native testing |
| `zig build` | Build system |
| `zig fmt` | Formatting |
| `zig-cache` | Cache |
| `gdb` / `lldb` | Debug |
| `valgrind` | Memory |
| `kcov` | Coverage |

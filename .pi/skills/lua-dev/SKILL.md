---
name: lua-dev
description: Lua development, testing, and quality assurance. Use for game scripting (Love2D, Roblox, WoW), embedded scripting, Neovim plugins, and OpenResty.
---

# Lua Development

## Top 10 Patterns

1. **Busted** — expressive BDD-style testing
2. **LuaUnit** — xUnit-style testing
3. **luacheck** — static analysis and linting
4. **LuaFormatter** — code formatting
5. **LuaCov** — code coverage
6. **Penlight** — standard library extensions
7. **LDoc** — documentation generation
8. ** EmmyLua / LuaCATS** — type annotations
9. **loadstring → load** — safe sandboxing
10. **coroutines** — cooperative multitasking

## Quality Gates

```bash
busted
luacheck .
luacov
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `busted` | BDD testing |
| `luaunit` | xUnit testing |
| `luacheck` | Lint |
| `luacov` | Coverage |
| `ldoc` | Docs |
| `luaformatter` | Format |

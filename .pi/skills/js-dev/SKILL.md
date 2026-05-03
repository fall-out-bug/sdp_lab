---
name: js-dev
description: Full-spectrum JavaScript/TypeScript development, testing, and quality assurance. Use for frontend, backend, tooling, and monorepo work across Node.js, Deno, Bun, and browser runtimes.
---

# JavaScript / TypeScript Development

## When to Use

- Any `.js`, `.ts`, `.jsx`, `.tsx`, `.mjs`, `.cjs` file changes
- Package management (npm, yarn, pnpm, bun)
- Build tooling (Vite, Webpack, Rollup, esbuild, tsc, SWC)
- Testing strategy decisions
- Linting / formatting setup
- Monorepo architecture (Turborepo, Nx, pnpm workspaces, Lerna)

## Runtimes

| Runtime | Use Case | Command |
|---------|----------|---------|
| Node.js | Production default | `node`, `npx` |
| Deno | Modern stdlib, permissions | `deno run`, `deno test` |
| Bun | Speed, bundling, testing | `bun run`, `bun test` |

## Top 10 Development Patterns

### 1. Type-First Development
Always define interfaces/types before implementation. Use `zod`, `valibot`, or `runtypes` for runtime validation.

### 2. Monorepo Workflows
```bash
# Turborepo
turbo run build test lint --filter=@scope/package

# pnpm workspaces
pnpm --filter @scope/pkg test

# Nx
nx test @scope/pkg --skip-nx-cache
```

### 3. Component-Driven Development
Use Storybook for isolated component development. Write stories before implementation.

### 4. E2E-First for Critical Paths
Playwright > Cypress > Selenium. Prefer `data-testid` over class selectors.

### 5. Mock Service Worker (MSW)
Use MSW for API mocking in tests AND development. Share handlers between test and dev.

### 6. Snapshot + Visual Regression
```bash
# Chromatic / Percy / Loki
npx chromatic --project-token=...
```

### 7. Property-Based Testing
```bash
# fast-check for JS
npm i -D fast-check
```

### 8. Performance Budgets
```bash
# bundlesize, bundlewatch
npx bundlesize
```

### 9. Contract Testing
```bash
# Pact for consumer-driven contracts
npx pact-cli verify
```

### 10. Mutation Testing
```bash
# Stryker for JS/TS
npx stryker run
```

## Testing Stack

| Layer | Tools |
|-------|-------|
| Unit | Vitest, Jest, Node test runner, Bun test |
| Integration | Vitest + MSW, Supertest, Playwright CT |
| E2E | Playwright, Cypress, WebdriverIO |
| Visual | Chromatic, Percy, Loki |
| Contract | Pact, Nock |
| Perf | Lighthouse CI, Web Vitals, k6 |
| A11y | axe-core, jest-axe, @axe-core/cli |
| Mutation | Stryker |

## Quality Gates

```bash
# Lint + format + typecheck + test
tsc --noEmit && eslint . && prettier --check . && vitest run
```

## Key Files

- `package.json` — scripts, dependencies, engines
- `tsconfig.json` — strict mode, paths, composite for monorepos
- `vitest.config.ts` — test config
- `playwright.config.ts` — e2e config
- `.eslintrc` — lint rules
- `turbo.json` / `nx.json` — monorepo orchestration

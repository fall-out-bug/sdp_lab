---
name: ux-testing
description: UX/UI testing, accessibility audits, visual regression, and performance validation. Use when verifying frontend user experience across devices and assistive technologies.
---

# UX / UI Testing

## When to Use

- Before declaring any frontend task "done"
- After component implementation
- Before release / PR merge
- When adding new user flows
- When updating design system tokens

## Top 10 Patterns

### 1. Automated Accessibility (a11y) First
Always run axe-core or Lighthouse a11y checks before manual QA.

### 2. Component-Driven + Visual Regression
Storybook stories → Chromatic / Percy for visual diffs on every PR.

### 3. E2E on Real Browsers
Playwright with `projects` for Chromium, Firefox, WebKit.

### 4. Responsive Testing Matrix
| Breakpoint | Device | Priority |
|------------|--------|----------|
| 320px | Mobile | P0 |
| 768px | Tablet | P0 |
| 1024px | Desktop | P0 |
| 1440px+ | Wide | P1 |

### 5. Performance Budgets
```bash
# lighthouse-ci
lhci autorun --config=lighthouserc.json
```
Budgets: LCP < 2.5s, CLS < 0.1, INP < 200ms

### 6. Keyboard Navigation Testing
Every interactive element must be reachable via Tab and have visible focus.

### 7. Screen Reader Validation
Test with NVDA (Windows), VoiceOver (macOS/iOS), TalkBack (Android).

### 8. Color Contrast Compliance
WCAG 2.1 AA: 4.5:1 for normal text, 3:1 for large text / UI components.

### 9. Motion Respect `prefers-reduced-motion`
```css
@media (prefers-reduced-motion: reduce) {
  * { animation-duration: 0.01ms !important; }
}
```

### 10. User Flow Recording + Replay
Use Playwright `trace.zip` or Session Replay (Sentry, LogRocket) for post-mortem.

## Testing Stack

| Concern | Tools |
|---------|-------|
| A11y | axe-core, Lighthouse, pa11y, WAVE |
| Visual | Chromatic, Percy, Loki, Applitools |
| E2E | Playwright, Cypress, WebdriverIO |
| Perf | Lighthouse CI, Web Vitals, k6 |
| Mobile | BrowserStack, Sauce Labs, Playwright device emulation |
| Interaction | Storybook test-runner, Testing Library |
| Form validation | Testing Library user-event |

## Quality Gates

```bash
# Pre-commit
npx playwright test --project=chromium --grep-invert "@slow"
npx lhci autorun
npx axe-core/cli http://localhost:3000

# CI
npx playwright test
npx chromatic --project-token=$CHROMATIC_TOKEN
```

## Critical Rules

- No UI task is "done" without a11y pass
- No component without a Storybook story
- No visual change without Chromatic / Percy review
- No release without Lighthouse performance budget check

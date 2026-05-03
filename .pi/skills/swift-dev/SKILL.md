---
name: swift-dev
description: Swift development, testing, and quality assurance. Use for iOS, macOS, watchOS, tvOS, server-side Swift (Vapor), and cross-platform tools.
---

# Swift Development

## Top 10 Patterns

1. **XCTest + swift-testing (new)** — modern test syntax
2. **SnapshotTesting** — snapshot tests for UI
3. **Quick + Nimble** — BDD style testing
4. **Sourcery** — code generation for mocks
5. **SwiftLint / SwiftFormat** — lint and format
6. **Periphery** — detect unused code
7. **Xcodebuild + xcpretty** — CLI builds
8. **SwiftUI Previews + ViewInspector** — testing SwiftUI
9. **Combine + Entwine** — reactive testing
10. **Vapor + XCTVapor** — server-side Swift testing

## Quality Gates

```bash
swift test --parallel
swiftlint lint --strict
swiftformat --lint .
xcodebuild test -scheme MyScheme -destination 'platform=iOS Simulator,name=iPhone 15' | xcpretty
```

## Key Tools

| Tool | Purpose |
|------|---------|
| XCTest / swift-testing | Testing |
| SnapshotTesting | UI snapshots |
| Quick / Nimble | BDD |
| SwiftLint / SwiftFormat | Lint/format |
| Periphery | Dead code |
| Sourcery | Codegen |
| ViewInspector | SwiftUI testing |
| XCTVapor | Server tests |

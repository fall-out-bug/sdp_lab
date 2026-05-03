---
name: dart-dev
description: Dart / Flutter development, testing, and quality assurance. Use for cross-platform mobile, web, desktop apps, and backend (Dart Frog, Serverpod).
---

# Dart Development

## Top 10 Patterns

1. **flutter_test / test** — widget and unit testing
2. **Widget testing** — `testWidgets`, `pump`, `finder`
3. **Golden tests** — `matchesGoldenFile` for UI regression
4. **Integration tests** — `integration_test` package
5. **Mockito / mocktail** — mocking
6. **Riverpod / Bloc** — state management testability
7. **flutter_lints / custom_lint** — static analysis
8. **dart format** — formatting
9. **dart doc** — documentation
10. **Flutter DevTools** — performance, memory, network profiling

## Quality Gates

```bash
flutter test --coverage
flutter analyze
flutter test integration_test/
flutter build apk --release
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `flutter test` | Testing |
| `test` / `flutter_test` | Frameworks |
| `mockito` / `mocktail` | Mocking |
| `integration_test` | E2E |
| `flutter_lints` | Lint |
| `dart format` | Format |
| `coverage` | Coverage |

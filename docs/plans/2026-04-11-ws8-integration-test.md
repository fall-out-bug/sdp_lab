# WS-8: Интеграционный тест

**Статус:** APPROVED (Council R2)
**Приоритет:** P0 (финальная верификация)
**Трудоёмкость:** 1ч
**Зависимости:** WS-5 (FIX-04), WS-6 (FIX-05), WS-7 (FIX-08)

## Цель
End-to-end верификация всего пайплайна после всех фиксов на реальных данных.

## Шаги
1. Собрать CLI: `go build ./cmd/sdp-strataudit`
2. Прогон на `/tmp/strataudit-test/` с конфигом `strataudit.yaml`
3. Верифицировать результаты

## Чеклист верификации
- [ ] `go test ./internal/strataudit/...` — все тесты зелёные
- [ ] `go vet ./internal/strataudit/...` — нет предупреждений
- [ ] CLI собирается без ошибок
- [ ] Отчёты создаются в `/tmp/strataudit-test/.strataudit/` (не в CWD)
- [ ] `report.json` содержит `entities: [...]`
- [ ] `report.json` содержит `traces: [...]`
- [ ] Findings на русском (`"Разрыв: ..."`, `"Сирота: ..."`)
- [ ] `finding.type` на английском (`"gap"`, `"orphan"`)
- [ ] Coverage >10% (vs 0-25% до фиксов)
- [ ] `report.html` открывается в браузере

## Commit
`test(strataudit): end-to-end integration test for v1.1 fixes`

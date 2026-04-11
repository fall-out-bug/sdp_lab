---
ws_id: 00-FFF-SS
feature_id: FFFF
status: backlog
priority: P2
size: M
depends_on: []
---

# 00-FFF-SS: [Короткое название задачи]

Feature: FFFF (sdplab-XXX)

## Goal

Одно–два предложения: что именно создаётся или исправляется, и зачем.
Без "мы", без пассивного залога. Конкретный результат.

## Beads

- sdplab-XXX

## Scope Files

Перечислить каждый файл явно. Для каждого файла — действие:
- `path/to/new/file.go` — создать
- `path/to/existing.go` — изменить (краткое что именно)
- `path/to/read_only.go` — прочитать (не менять)
- `path/to/test_file_test.go` — создать тест

## Acceptance Criteria

Каждый пункт — проверяемый факт, не намерение.
Плохо: "добавить поддержку streaming"
Хорошо: "go test ./internal/llmclient/... -run TestStream -race PASS"

- [ ] [Что именно проверяется — команда или наблюдаемый результат]
- [ ] `go build ./...` успешен
- [ ] `go test ./path/to/package/... -race` PASS
- [ ] `go vet ./...` чистый

## Out of Scope

Явно перечислить что НЕ входит в эту задачу, чтобы агент не расширял scope.

- [Что именно исключено]

## Implementation Notes

Контекст для агента: ссылки на дизайн-документ, ключевые типы, паттерны.
Не повторять acceptance criteria. Не писать код здесь — код в плане.

- Подробные шаги: `docs/plans/YYYY-MM-DD-feature-name-plan.md`, раздел WS-SS
- [Ключевой тип или интерфейс с кратким описанием]
- [Паттерн или ограничение, которое агент должен знать]

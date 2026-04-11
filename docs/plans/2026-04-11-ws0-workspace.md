# WS-0: Создание workspace

**Статус:** DONE
**Приоритет:** P0
**Трудоёмкость:** 5 мин
**Зависимости:** нет

## Цель
Изолированный workspace (git worktree) от `main` для безопасной реализации всех фиксов.

## Шаги
1. Создать worktree `strataudit-v1.1` от `main`
2. `go build ./...` — убедиться что baseline собирается
3. `go test ./internal/strataudit/...` — baseline тесты зелёные

## Приёмка
- Worktree создан на ветке `worktree-strataudit-v1.1`
- Build и тесты проходят без ошибок

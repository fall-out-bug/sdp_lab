# Happy Path: Cold Start

> **Ситуация:** Новый проект, новая сессия агента — никакого контекста. Как начать без хаоса?

> **Status:** operator/lab recipe, not the friendly first-run Toolkit path.
> If you are trying SDP in your own repo for the first time, start with
> [../START_HERE.md](../START_HERE.md) and [../QUICKSTART.md](../QUICKSTART.md).
> Commands such as `sdp-up` and `sdp-harness` are lab/operator surfaces, not the
> first thing an external user should run.

Cold start = первые действия агента в неизвестном или свежем контексте.

---

## Для агентов: Cold Start Protocol

Если ты агент и только что открыл этот репозиторий — выполни эти шаги в порядке:

### Шаг 1: Ориентация (60 секунд)

```bash
git status --short --branch          # где я?
git log --oneline -5                 # что происходило?
ls                                   # что тут есть?
```

Прочитай в таком порядке:
1. `VISION.md` — что такое SDP (1 мин)
2. `AGENTS.md` — как тут работают (2 мин)
3. `docs/ARCHITECTURE.md` — компонентная карта (1 мин)

---

### Шаг 2: Найти работу

```bash
scripts/beads_transport.sh fetch     # синхронизировать Beads
bd ready                             # что готово к работе?
```

Если `bd ready` пуст:
```bash
bd list --status=open                # что вообще открыто?
bd blocked                           # что заблокировано и почему?
```

---

### Шаг 3: Понять конкретную задачу

```bash
bd show <id>                         # прочитай issue целиком
# Найди workstream файл из секции issue
cat docs/workstreams/backlog/00-FFF-SS.md
```

Прочитай:
- Goal (что это?)
- Acceptance Criteria (что значит "done"?)
- Out of Scope (что НЕ делать?)
- depends_on (есть ли незакрытые blocker'ы?)

**Если есть blocker:** `bd blocked` → найди что блокирует → не начинай заблокированную задачу.

---

### Шаг 4: Проверить контекст фазы

**Discovery задача?**
```bash
cat docs/plans/2026-*-discovery-*.md 2>/dev/null
ls docs/discovery/ 2>/dev/null
```
→ Читай [docs/guides/agent-discovery.md](../guides/agent-discovery.md)

**Delivery задача?**
```bash
ls .sdp/sessions/ 2>/dev/null       # есть ли незавершённые сессии?
```
→ Читай [docs/guides/agent-delivery.md](../guides/agent-delivery.md)

---

### Шаг 5: Claim и начинай

```bash
bd update <id> --claim
git checkout main && git pull
git checkout -b <branch-name>
```

Только после claim — начинай работу.

---

## Для операторов: Cold Start нового проекта

Если ты человек и запускаешь SDP с нуля:

### Минимальная инициализация

```bash
# В корне нового проекта
sdp-up

# Проверить что всё работает
sdp-ready
```

### Первая идея → первый workstream

```bash
# Опционально: Discovery если идея нетривиальная
sdp discover "что хочу построить"

# Создать первый feature
bd create \
  --title="Foundation: scaffold + CI" \
  --description="Первый рабочий scaffold" \
  --type=feature --priority=0

# Проверить
bd ready
```

### Запуск первого агента

```bash
bd update <id> --claim
sdp-harness new --card-id=<id> --project=<name>
sdp-harness run --session=<id> --prompt="..."
```

---

## Cold Start Checklist

```
[ ] git status + log — понять состояние репо
[ ] Прочитал VISION.md
[ ] Прочитал AGENTS.md  
[ ] bd ready — знаю что готово к работе
[ ] Выбрал одну задачу (не несколько)
[ ] bd show <id> — прочитал acceptance criteria
[ ] Нет незакрытых blocker'ов
[ ] bd update <id> --claim — claim'нул задачу
[ ] git checkout -b <branch>
```

---

## Если что-то непонятно

**Не угадывай.** Один конкретный вопрос лучше чем неправильная реализация.

Сначала проверь документацию:
- `docs/TERMS.md` — глоссарий
- `docs/ARCHITECTURE.md` — компоненты
- `docs/phases/DISCOVERY.md` / `DELIVERY.md` — фазы
- `docs/reference/components.md` — каталог

Если не нашёл → задай вопрос оператору.

---

*Полная карта навигации: [docs/ARCHITECTURE.md](../ARCHITECTURE.md)*

# F136 — Peer Memory Foundation (Horizon 1)

**Status:** design
**Date:** 2026-04-20
**Owner:** Andrei
**Epic:** `F136`
**Horizon:** H1 (Q2 2026). H2/H3 parked — revisit 2026-07-06 / 2026-10-05.
**Prior art:** F122 (code index), F126 (MCP), F129 (autonomy regression), beads/Dolt.

---

## Problem

SDP положил в фундамент три слоя памяти:

- **semantic** (F122 code index) — что есть в коде,
- **procedural** (prompts / skills / patterns) — как делать вещи,
- **working** (manifest.md + context window) — что сейчас в голове.

Отсутствует четвёртый слой — **episodic**: что случилось в прошлых сессиях, кто что решил и почему. Каждый агент (и человек) входит в репо холодным, и *решения теряются между сессиями*.

Рынок (Devin, Windsurf, Cursor) строит «operator + autonomous cloud agents». Академия (ChatCollab, HULA, CrafTeam) показала, что peer-модель работает, но не продуктизировала её. **Пустой квадрат на апрель 2026 — production-ready peer memory, где attribution first-class и люди с агентами пишут в один лог.**

H1 цель: доказать в реальном использовании, что actor-aware episodic memory ценен. Без философии, без полного peer framework. Минимальная рабочая вещь.

## Non-Goals (H1)

Явно вне скоупа — перенесено в H2/H3:

- ❌ Shared mental model artifact (`.sdp/team/context.md`)
- ❌ Symmetric monitoring (agent flag'ит human)
- ❌ Trust-tier inference и challenge-протокол
- ❌ Dolt sync / team federation
- ❌ Real-time presence
- ❌ Role negotiation

## Success Criterion (binary)

Через месяц использования в `sdp_lab`: нашёл ли я через `sdp memory query "<question>"` хоть один реальный контекст, который раньше был бы потерян между сессиями. Да/нет.

Если **да** — открываем H2 в июле.
Если **нет** — park, возможно, проблема в модели триггеринга, не в инфре.

## Architecture

```
┌────────────────────────────────────────────────────────┐
│  WRITE path                                            │
│    human:  sdp memory log --role=architect "…"         │
│    agent:  internal/memory.Append(entry) via agentloop │
│    hook:   session-close → compact → summary.md        │
└────────────────────────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────────┐
│  STORE                                                 │
│    .sdp/memory/                                        │
│      events.jsonl        ← append-only                 │
│      summaries/YYYY-MM-DD.md   ← LLM-compacted         │
│      index.db               ← FTS5 over events+summaries│
└────────────────────────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────────┐
│  READ path                                             │
│    CLI:  sdp memory query "…"                          │
│    MCP:  sdp://memory/recent, sdp://memory/query/{q}   │
│    agent cold start: load last N summaries + manifest  │
└────────────────────────────────────────────────────────┘
```

## Schema v1

Одна строка JSONL = одно событие. Append-only.

```go
type MemoryEntry struct {
    ID          string    `json:"id"`           // ulid
    Timestamp   time.Time `json:"ts"`
    SessionID   string    `json:"session_id"`
    Author      string    `json:"author"`       // "andrei", "claude-sonnet-4-6", "codex-gpt-5"
    AuthorType  string    `json:"author_type"`  // "human" | "agent"
    Role        string    `json:"role"`         // "architect" | "implementer" | "reviewer" | "qa" | "operator"
    Type        string    `json:"type"`         // "decision" | "discovery" | "handoff" | "question" | "note"
    Content     string    `json:"content"`      // freeform NL
    Links       []string  `json:"links"`        // beads_id, chunk_id, file:line, pr_url
    Tags        []string  `json:"tags"`         // "auth", "f136", etc.
}
```

**Что важно и почему:**

- `author` + `author_type` — **attribution first-class**. Это единственный нетривиальный архитектурный выбор H1; всё остальное — имплементация.
- `role` не то же что `author_type`. Человек может играть role=reviewer, агент может играть role=architect. Роли пересекают тип.
- `type` — минимальный набор для дифференциации поиска («что мы *решили* vs что *обнаружили*»).
- `links` — дешёвая проекция на существующий граф (beads, F122 chunks, PR).
- `session_id` — группировка для компакции.
- **Нет** `trust_tier` в H1 — это H2 territory. В H1 все записи равны.

## Write Triggers (H1 — три категории)

**Явные (человек пишет):**
- `sdp memory log --type=decision "отказались от goroutine pool: race на shutdown"`
- `sdp memory log --type=handoff "передаю claude, блокируется на flake в TestX"`

**Агентные (агент пишет через `internal/memory.Append`):**
- В конце каждого `agentloop` phase transition — автоматически, один entry на фазу
- При claim issue (`bd update --claim`) — handoff entry
- При закрытии issue — decision entry с финальным rationale

**Автоматический (hook):**
- `session-close` hook (`.claude/hooks/session-stop-memory.sh`) — если `events.jsonl` за сессию не пуст, предложи компакцию в summary

**Что НЕ триггерит в H1:** file-save, git commit, PR review. Слишком шумно, overengineering.

## Compaction

При закрытии сессии:

1. Читаем `events.jsonl` отфильтрованный по `session_id`
2. Если ≥5 записей — зовём LLM (модель по умолчанию — та же что в `agentloop`) на one-shot суммаризацию
3. Пишем `summaries/YYYY-MM-DD-<session_id>.md` с frontmatter:
   ```yaml
   ---
   session_id: ...
   started: ...
   ended: ...
   participants: [andrei, claude-sonnet-4-6]
   ---
   ```
4. `events.jsonl` **не** удаляем — summary это read-optimized view, events — ground truth

Garbage collection: `sdp memory compact --older-than=90d` опционально архивирует old events в `.sdp/memory/archive/`.

## Read Path

Три команды:

```bash
sdp memory query "<question>"           # FTS5 over events + summaries + author filter
sdp memory recent [--author=X] [--session=Y]
sdp memory show <entry_id>              # full entry + linked chunks/issues
```

**Search backend v1:** простой FTS5 SQLite над events + summaries. **Не** используем vector search пока — pointless для небольших объёмов (<10K entries = меньше чем у одного разработчика за год). Если H2 пойдёт — добавим sqlite-vec (инфраструктура уже в F122).

**MCP resources (F126 extension):**
- `sdp://memory/recent` → JSON array последних 20 entries
- `sdp://memory/query?q=...` → поисковые результаты
- `sdp://memory/session/{id}` → полная сессия

Write path через MCP **сознательно не делаем** в H1 (см. анализ в brainstorm). MCP = read. Writes через CLI или Go API.

## Integration Points

| Существующее | Что меняем |
|---|---|
| `internal/agentloop/` | добавить `memory.Append` в phase transitions |
| `internal/sdpctx/` или аналог | session_id из context |
| `cmd/sdp/` | `cmd_memory.go` — новая subcommand |
| `.claude/hooks/` | новый `session-stop-memory.sh` |
| MCP server (F126) | регистрация `sdp://memory/*` resources |
| `bd` | не трогаем — memory **не заменяет** beads. beads = задачи, memory = события |

## Storage & Privacy

- `.sdp/memory/` — **в `.gitignore` по умолчанию**. Локальный state.
- Summaries/markdown можно опционально коммитить (`sdp memory share --summary=2026-04-20`), но это opt-in в H1.
- Никакой автоматической отправки наружу. H2/H3 подумают про sync.
- No secrets: regex-scanner перед записью блокирует запись при совпадении `(password|token|api_key|secret|BEGIN (RSA|OPENSSH) PRIVATE KEY)` и типовых env-var форматах.

## Testing

1. `store_test.go` — append, query, filter by author/type, concurrent appends
2. `compact_test.go` — mock LLM, verify summary structure
3. `cli_test.go` — golden files для `sdp memory query` output
4. Integration: запустить agentloop на mock-issue, verify entries появились
5. MCP resource tests — запрос через MCP клиент возвращает JSON

## Package Layout

```
internal/memory/
├── types.go          # MemoryEntry, Author, Role enums
├── store.go          # JSONL append/read, session scoping
├── store_test.go
├── query.go          # FTS5 search, filters
├── query_test.go
├── compact.go        # LLM summarization
├── compact_test.go
├── secrets.go        # pre-write secret scanner
└── secrets_test.go

cmd/sdp/
└── cmd_memory.go     # subcommand wiring

.claude/hooks/
└── session-stop-memory.sh
```

## Workstreams → Beads

H1 декомпозируется в 6 work items:

1. **F136-01** — Design finalization + schema freeze (this doc + review)
2. **F136-02** — `internal/memory/` core: types, JSONL store, secret scanner
3. **F136-03** — `sdp memory` CLI (log, query, recent, show, compact)
4. **F136-04** — Session-close hook + agentloop auto-capture integration
5. **F136-05** — MCP resource exposure (F126 extension, read-only)
6. **F136-06** — Documentation + AGENTS.md patch + one-month dogfood period

**Dependencies:**
- F136-01 → F136-02
- F136-02 → {F136-03, F136-04, F136-05} (parallel)
- {F136-03, F136-04, F136-05} → F136-06
- F136-06 → F136 epic close

**Out of scope (explicit parking):**

| Item | Horizon | Revisit |
|---|---|---|
| Shared mental model artifact | H2 | 2026-07-06 |
| Symmetric monitoring / challenge protocol | H2 | 2026-07-06 |
| Trust tiers + inference | H2 | 2026-07-06 |
| Dolt sync / team federation | H3 | 2026-10-05 |
| Role negotiation / adaptive allocation | H3 | 2026-10-05 |
| Real-time presence | H3 | 2026-10-05 |
| Vector search over memory | H2 conditional | 2026-07-06 |

## Risks

| Risk | Mitigation |
|---|---|
| Humans не пишут — memory становится agent-only | Session-close prompt с низким трением («добавить одну строку?») |
| Compaction стоит денег на LLM | Только при ≥5 entries; fallback на format-based summary без LLM |
| Секреты утекают в memory | Regex-scanner перед append; тесты на типовые паттерны |
| Overengineering для локального use | JSONL + FTS5, без vector, без graph. Если мало — добавим в H2 |
| Спутают beads и memory | AGENTS.md строго: beads = задачи, memory = события/решения |

## Open Questions (неблокирующие H1)

- Нужно ли различать `agent` от `skill`/`tool` в author_type? Скорее всего нет, это H2.
- Какой author_id для cross-harness agents (Claude Code vs Codex vs OpenCode)? — договоримся `<harness>-<model>` но это H1-02 implementation detail.
- Как обрабатывать параллельные сессии одного человека? `session_id` из env var, уникальный per shell.

## Exit Criteria (H1 complete)

- [ ] F136-01..06 закрыты в beads
- [ ] `sdp memory log|query|recent|show|compact` работают
- [ ] agentloop пишет entries автоматически
- [ ] MCP resources доступны
- [ ] Один месяц dogfood в sdp_lab
- [ ] Binary success check (см. Success Criterion) выполнен с ответом «да»/«нет»
- [ ] H2 review decision зафиксирован в beads

**Revisit H2/H3 gate:** 2026-07-06 (первый понедельник июля, +11 недель от старта).

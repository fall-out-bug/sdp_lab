# Evidence + Enforcement: план перехода к реальности

**Дата:** 2026-02-25  
**Контекст:** Harsh engineer audit, Phase 0 enforcement audit  
**Цель:** evidence и enforcement действительно блокируют merge, а не только документированы

---

## 1. Текущее состояние (почему не работает)

| Компонент | Ожидание | Реальность |
|-----------|----------|------------|
| **evidence** | Пишется в `.sdp/evidence/FXXX.json`, попадает в PR | `.sdp/*` в .gitignore → evidence не коммитится |
| **evidence-gate** | Валидирует evidence, блокирует при ошибке | Ищет evidence в `git diff` → всегда пусто → exit 0 |
| **checkpoint** | Используется scope-gate | `.sdp/checkpoints/` в .gitignore → не коммитится |
| **scope-gate** | Блокирует при scope violation | `|| echo "WARN"` → никогда не падает |
| **branch protection** | evidence-gate, scope-gate required | F030 в Backlog; возможно не настроено |

**Итог:** evidence и checkpoint не доходят до CI. Gates либо не запускаются, либо не блокируют.

---

## 2. Что нужно сделать

### 2.1 Evidence должен коммититься

**Проблема:** `.sdp/*` в .gitignore, evidence не попадает в репо.

**Варианты:**

| Вариант | Действие | Плюсы | Минусы |
|---------|----------|-------|--------|
| **A: Commit evidence** | Добавить `!.sdp/evidence/` в .gitignore | Просто, gate сразу видит файлы | Шум в diff, размер (JSON) |
| **B: Commit только для feature PR** | Policy: feature/* ветки должны иметь evidence | Меньше шума для docs-only | Сложнее: как определить feature? |
| **C: CI artifact вместо commit** | evidence создаётся в CI, gate валидирует артефакт | Нет шума в репо | Нужен другой flow: agent не пишет, CI генерирует |

**Рекомендация:** **A** — добавить `!.sdp/evidence/` в .gitignore. Минимальный шаг. Evidence уже пишется orchestrate; нужно только коммитить.

**Действия:**
1. В `.gitignore`: добавить `!.sdp/evidence/`
2. В @build / orchestrate flow: после `sdp-orchestrate --advance` — `git add .sdp/evidence/` и commit (или amend)
3. Документировать в AGENTS.md: "Evidence must be committed with the PR"

---

### 2.2 evidence-gate должен блокировать

**Проблема:** При отсутствии evidence в diff gate делает "skip" и exit 0.

**Логика:**
- Если PR трогает `internal/`, `cmd/`, `docs/workstreams/` — это feature-изменения → evidence обязателен
- Если evidence в diff — валидировать; при ошибке — exit 1
- Если evidence обязателен, но его нет в diff — exit 1

**Действия:**
1. В `evidence-gate` job: если `git diff` содержит `internal/`, `cmd/`, или `docs/workstreams/backlog/`, и при этом нет `.sdp/evidence/*.json` в diff → `echo "ERROR: feature PR requires evidence"`; `exit 1`
2. Убрать "exit 0" при пустом EVIDENCE_FILES — заменить на проверку "feature PR?" и при да — требовать evidence

---

### 2.3 scope-gate должен блокировать

**Проблема:** `|| echo "WARN"` — guard никогда не падает.

**Действия:**
1. Убрать `|| echo "WARN"` — если `go run ./cmd/sdp-guard --ws "$ws"` возвращает non-zero, job должен fail
2. **Checkpoint в diff:** scope-gate сейчас срабатывает только при checkpoint в diff. Checkpoint тоже gitignored. Нужно ли коммитить checkpoint?
   - **Вариант 1:** Коммитить checkpoint (`!.sdp/checkpoints/`) — тогда scope-gate увидит его
   - **Вариант 2:** scope-gate триггерится по-другому: если в diff есть `docs/workstreams/backlog/` или `internal/`, `cmd/` — определить WS из branch name или из изменённых файлов и проверить scope
   - **Вариант 3:** scope-gate всегда запускается для feature PR; WS определяется из branch (feature/F053-*) или из checkpoint если есть

**Рекомендация:** Коммитить checkpoint (`!.sdp/checkpoints/`) — он уже используется orchestrate, и scope-gate из него берёт WS list. Консистентно с evidence.

---

### 2.4 Flow: кто коммитит evidence и checkpoint

**Текущий flow:**
1. Agent делает @build, коммит кода
2. User/agent запускает `sdp-orchestrate --advance --result $(git rev-parse HEAD)`
3. Advance пишет checkpoint и evidence
4. Checkpoint и evidence остаются uncommitted (gitignore)

**Новый flow:**
1. Agent делает @build, коммит кода
2. `sdp-orchestrate --advance --result $(git rev-parse HEAD)` — пишет checkpoint, evidence
3. **Agent коммитит:** `git add .sdp/evidence/ .sdp/checkpoints/ && git commit -m "F053: evidence + checkpoint"` (или amend к предыдущему)
4. Push

**Изменения в skills:**
- @build: после успешного build и advance — добавить шаг "commit evidence and checkpoint"
- Или: post-build hook в pipeline-hooks.yaml, который делает `git add .sdp/evidence/ .sdp/checkpoints/`
- Или: sdp-orchestrate --advance сам делает commit (но это смешивает ответственность)

**Рекомендация:** Явный шаг в @build skill: "After sdp-orchestrate --advance: git add .sdp/evidence/ .sdp/checkpoints/; git commit --amend --no-edit || git commit -m 'FXXX: evidence'"

---

### 2.5 Branch protection

**Действия:**
1. В GitHub: Branch protection rule для master
2. Required status checks: `build-test`, `evidence-gate`, `policy-gate`
3. scope-gate: включить в required, если он теперь блокирует; или оставить advisory на первом этапе
4. "Do not allow bypassing" — включить

**F030:** Закрыть или обновить — branch protection как done.

---

## 3. Порядок внедрения

| # | Действие | Effort | Блокер |
|---|----------|--------|--------|
| 1 | .gitignore: `!.sdp/evidence/`, `!.sdp/checkpoints/` | 5 min | — |
| 2 | evidence-gate: require evidence для feature PR (internal/, cmd/, docs/workstreams/) | 30 min | — |
| 3 | scope-gate: убрать `|| echo "WARN"` | 5 min | — |
| 4 | @build skill: commit evidence + checkpoint после advance | 30 min | 1 |
| 5 | Branch protection (F030) | 15 min | — |
| 6 | Тест: PR без evidence → CI fail | 15 min | 1, 2 |
| 7 | Тест: PR с invalid evidence → CI fail | 15 min | 1, 2 |
| 8 | Тест: scope violation → CI fail | 15 min | 1, 3 |

---

## 4. Риски и митигации

| Риск | Митигация |
|------|-----------|
| Evidence файлы большие | Оставить только .sdp/evidence/; при необходимости — .gitattributes для diff |
| Существующие PR сломаются | Ввести по фазам: сначала 1–3, проверить; потом 4–5 |
| Agent не коммитит evidence | Чёткий AC в @build; post-build hook как fallback |
| Checkpoint содержит чувствительные данные | Проверить содержимое; при необходимости — exclude части |

---

## 5. Критерий успеха

- PR с изменениями в `internal/` или `cmd/` без `.sdp/evidence/*.json` в diff → **evidence-gate fail**
- PR с invalid evidence (broken JSON, missing sections) → **evidence-gate fail**
- PR со scope violation (guard fail) → **scope-gate fail**
- Branch protection: merge без passing CI → **blocked**

---

## 6. Следующие шаги

1. ~~**Создать workstream(s)**~~ — F055 "Evidence Enforcement Reality" оформлен отдельно (00-055-01 … 00-055-03)
2. **Реализовать** — после завершения F054: `@build 00-055-01` → 00-055-02 → 00-055-03
3. **Обновить docs** — MANIFESTO, getting-started — actual behavior

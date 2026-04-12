# Анализ состоятельности SDP как протокола

> **Дата:** 2026-02-24
> **Вопрос:** Нужен ли SDP как протокол? Решает ли он реальную проблему? Не переизобретаем ли мы велосипед?

---

## Что делают другие (рынок, февраль 2026)

### Workflow для coding agents

| Подход | Кто | Что делает | Формальность |
|--------|-----|------------|-------------|
| **plan.md + annotation** | Boris Tane, indie devs | research.md → plan.md → аннотации → implement | Минимальная: markdown + git |
| **CLAUDE.md / AGENTS.md** | Большинство команд | Один файл с конвенциями проекта | Конвенции, не протокол |
| **Deliberate Agentic Dev** | Matt Hulme (GitHub) | Plan → Build → Ship с чекпоинтами | Фазы, но без evidence |
| **AGENT-ZERO** | Open source | PLAN → BUILD → DIFF → QA → APPROVAL → APPLY → DOCS | State machine + Memory Bank |
| **Stripe Minions** | Stripe (internal) | Blueprint (directed graph) + devbox + Toolshed MCP | Формальная, но закрытая |
| **Open SWE** | LangChain | Planning + human approval + parallel execution | Deprecated в 2026 |
| **SDP** | Мы | Features → Workstreams → build → review → PR → CI | Самая формальная из всех |

**Вывод:** Никто кроме Stripe не использует формальную декомпозицию на уровне features/workstreams. Большинство используют plan.md или CLAUDE.md. SDP — самый формальный подход на рынке.

### Evidence / attestation для agent output

| Инструмент | Что делает | Слой | Статус |
|------------|------------|------|--------|
| **EPI** (epilabs.org) | `.epi` файлы = ZIP с execution steps + Ed25519 подпись | Inference trace (промпты, ответы, токены) | Production v2.4, `pip install epi-recorder` |
| **WorkProof** | Firecracker VM + blockchain attestation | Execution verification | Beta, crypto-focused |
| **ProofTrail** | Tamper-proof receipts | Agent actions | Early stage |
| **GLACIS** | Continuous attestation, ~5ms overhead | LLM API governance | Enterprise (finance, healthcare) |
| **TrustPlane** | Certified Writes + Action Certificates | Enterprise AI writes | Fortune 50 deployed |
| **VET** (paper) | Cryptographic execution traces | Host-independent verification | Academic |
| **SDP** | 9-section evidence envelope | Development process (scope, review, risk, trace) | In-toto migration done |

**Вывод:** Evidence для AI agents — растущий рынок. Но существующие инструменты покрывают **inference layer** (что LLM сказал) и **execution layer** (что код сделал). Никто не покрывает **development process layer** (scope compliance, review verdict, risk notes, AC mapping).

---

## Три слоя SDP — честная оценка

### Слой 1: Protocol (features/workstreams)

**Проблема:** Формальная декомпозиция feature → workstreams с AC, scope files, dependencies.

**Нужно ли это?**
- **Для human-in-the-loop** (Boris Tane pattern): **Нет.** plan.md + annotation достаточно. Человек и так контролирует scope и качество.
- **Для semi-autonomous** (sdp-orchestrate): **Да.** Orchestrator нужны boundary (scope files), criteria (AC), ordering (dependencies). Без них orchestrator не знает что делать.
- **Для fully autonomous** (K8s dream): **Обязательно.** Stripe's Blueprint — это тот же формализм, только в виде directed graph.

**Вывод:** Workstreams нужны, но **только для автономных flow**. Для ручной работы — overkill. Нужно два режима: light (plan.md) и full (workstreams).

### Слой 2: Evidence (attestation)

**Проблема:** Доказать, что агент: спланировал, остался в scope, прошёл тесты, был отревьюен, зафиксировал риски.

**Нужно ли это?**
- **EPI** покрывает inference trace — промпты, ответы, токены. Это **не то же самое** что development process.
- **GLACIS/TrustPlane** покрывают LLM API governance — policy enforcement на уровне вызовов к модели. Тоже **не то**.
- **SDP** покрывает уникальный слой: scope compliance, review verdict, risk notes, AC-to-evidence mapping, boundary (declared vs observed). **Этого нет ни у кого.**

**Вывод:** Evidence для development process — реальная и незанятая ниша. Но формат должен быть стандартным (in-toto), а генерация — автоматической (CI observer).

### Слой 3: Skills/Commands (prompts)

**Проблема:** Структурированные промпты для агентов (@build, @review, @deploy).

**Нужно ли это?**
- CLAUDE.md / AGENTS.md — стандарт для project conventions.
- Skills — это расширение CLAUDE.md для специализированных workflow.
- Рынок подтверждает: skills используются повсеместно.

**Вывод:** Skills — нормальная часть экосистемы. Но они должны быть **тонкими** (50-100 строк) и не дублировать то, что IDE делает нативно.

---

## Паттерн Boris Tane vs SDP

| Аспект | Boris Tane | SDP |
|--------|------------|-----|
| Планирование | plan.md + аннотации | Workstream files с AC и scope |
| Кто контролирует | Человек (аннотации) | Orchestrator (state machine) |
| Evidence | Нет | 9-section envelope |
| Scope enforcement | Человек проверяет | sdp-guard (git diff vs scope) |
| CI | Стандартный | evidence-gate + scope-gate + policy-gate |
| Target user | SWE с coding agent | Команда с автономными агентами |

**Ключевой инсайт Boris Tane:** "I want implementation to be boring." Протокол должен быть **невидимым**. Пользователь видит план и прогресс. Workstreams, evidence, scope — это plumbing.

---

## Что SDP должен стать

### Невидимый протокол

Пользователь:
```
@oneshot "Add JWT authentication"
```

Что видит:
```
Planning... 3 workstreams identified
Building 1/3: auth middleware... done
Building 2/3: token service... done  
Building 3/3: tests... done
Reviewing... APPROVED (0 P0/P1, 2 P3)
PR #42 created, CI running...
CI GREEN — done
```

Что происходит внутри (невидимо):
- Workstreams с AC и scope files создаются автоматически из плана
- sdp-guard проверяет scope после каждого build
- Evidence генерируется автоматически CI observer'ом
- OPA policies проверяют compliance при merge
- in-toto attestation подписывается Sigstore

**Пользователь никогда не пишет workstream файлы вручную.** Они генерируются из плана. Пользователь видит прогресс и результат. Evidence — артефакт CI, не артефакт человека.

### Два режима

**Light mode (Boris Tane):**
- plan.md → annotate → implement
- Evidence: auto-generated в CI (minimal: changed files, test results, scope check)
- Enforcement: CI gates + branch protection
- Для: human-in-the-loop teams

**Full mode (autonomous):**
- Feature → workstreams → sdp-orchestrate → evidence → PR
- Evidence: full 9-section attestation
- Enforcement: CI + OPA + runtime governance
- Для: teams running autonomous agents (future: K8s dream)

---

## Что нужно скрыть от пользователя

| Пользователь видит | Система делает невидимо |
|--------------------|-----------------------|
| План (план.md или описание фичи) | Декомпозиция на workstreams с AC и scope |
| Прогресс (building 1/3, reviewing...) | State machine transitions, checkpoint saves |
| Результат (PR created, CI green) | Evidence generation, attestation signing |
| Ошибки (scope violation, P0 finding) | Guard checks, policy evaluation |

## Что нужно показать

| Кому | Что |
|------|-----|
| SWE (автору PR) | Прогресс, результат, ошибки |
| Reviewer (человеку) | Evidence summary, scope compliance, risk notes |
| Audit/compliance | Full attestation (in-toto signed) |
| CI/CD | Gate results (pass/fail) |

---

## Итоговая оценка

| Вопрос | Ответ |
|--------|-------|
| Протокол (features/workstreams) нужен? | **Да, для автономных агентов. Нет, для ручной работы. Нужно два режима.** |
| Evidence переизобретает EPI? | **Нет. EPI = inference trace. SDP = development process attestation. Разные слои.** |
| Skills нужны? | **Да, как часть CLAUDE.md/AGENTS.md экосистемы.** |
| Уникальная ценность? | **Development-process attestation (scope, review, risk, AC) — ниша, которую не занял никто.** |
| Должен быть невидимым? | **Да. Протокол = plumbing. Пользователь видит план, прогресс, результат.** |

---

## References

- [Boris Tane: How I Use Claude Code](https://boristane.com/blog/how-i-use-claude-code/) — plan.md + annotation pattern
- [AGENTS.md standard](https://justaddai.net/standards/agents-md/) — universal AI agent instructions
- [Deliberate Agentic Development](https://github.com/Matt-Hulme/deliberate-agentic-development) — Plan → Build → Ship
- [EPI](https://www.epilabs.org/) — evidence packaged infrastructure (inference trace)
- [WorkProof](https://www.workproof.run/) — blockchain verification for agent work
- Task decomposition for coding agents: 58% faster with structured decomposition
- SEW (arXiv 2505.18646): Self-evolving workflows, +33% on benchmarks
- AgentMesh (arXiv 2507.19902): Planner → Coder → Debugger → Reviewer

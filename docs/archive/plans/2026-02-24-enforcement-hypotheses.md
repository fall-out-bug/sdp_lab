# Enforcement Hypotheses: What We Can Learn From Others

> **Date:** 2026-02-24
> **Trigger:** Phase 0 audit showed 93% of controls are advisory, not enforced
> **Goal:** Сформулировать гипотезы, найти аналоги в индустрии, продумать тестирование

---

## Мы не одиноки

Проблема «инструменты есть, enforcement нет» решена в нескольких дисциплинах. Мы изобретаем велосипед, игнорируя готовые решения.

### Прямые аналоги

| Дисциплина | Проект / Стандарт | Что делает | Прямое соответствие в SDP |
|------------|-------------------|------------|---------------------------|
| **Supply chain security** | **SLSA** (Supply-chain Levels for Software Artifacts) | 4 уровня гарантий: от «кто собрал» до «невозможно подделать». Provenance attestations подписаны криптографически. | Evidence envelope = attestation. Но SLSA требует **верификацию при merge**, а не advisory. |
| **Supply chain security** | **in-toto** | Фреймворк: каждый шаг pipeline создаёт подписанный link-файл. `in-toto-verify` проверяет что ВСЕ шаги выполнены. Нет link = нет merge. | Наша цепочка beads → commit → evidence = in-toto layout. Но у нас нет `verify` на merge path. |
| **Supply chain security** | **Tekton Chains** | K8s-контроллер: наблюдает за TaskRun, **автоматически** генерирует provenance attestation + подпись. Разработчик НЕ МОЖЕТ забыть — attestation создаётся автоматически. | Наш evidence создаётся агентом вручную. Tekton Chains создаёт attestation автоматически. |
| **Supply chain security** | **Sigstore / cosign** | Keyless signing в CI. Cryptographic proof что артефакт создан в определённом CI run. Нет подписи = нет deploy. | Мы можем подписывать evidence в CI. Нет подписи = нет merge. |
| **Policy-as-code** | **OPA / Rego** | Декларативные политики. CI вызывает `opa eval`; exit 1 = merge blocked. Политики версионированы рядом с кодом. | quality-gates.md — это политики, но в markdown, не в исполняемом формате. |
| **K8s admission** | **Kyverno** | Admission controller: ресурс не создаётся в кластере, если не соответствует policy. Bypass невозможен — проверка на уровне API server. | Аналог: merge как «admission» в master. Policy проверяется при merge, не при commit. |
| **Git enforcement** | **Pre-receive hooks** (server-side) | Серверные хуки. `--no-verify` НЕ работает. Единственный способ обойти — доступ к серверу. | Наши хуки — client-side (pre-push, Stop hook). Client-side = обходимы. |
| **Agent governance** | **MI9** (arXiv 2508.03858) | 6 компонентов runtime governance: FSM-conformance engine, drift detection, graduated containment. 99.81% detection rate. | Наш `sdp orchestrate` — зачаток FSM-conformance. Но без drift detection и containment. |
| **Agent governance** | **AgentSpec** (arXiv 2503.18666, ICSE 2026) | DSL для runtime constraints: trigger → predicate → enforcement. 90%+ prevention rate. Millisecond overhead. | Мы можем описать constraints декларативно (не в промптах) и enforce в runtime. |
| **Agent governance** | **A2AS** (arXiv 2510.13825) | «HTTPS for AI agents»: behavior certificates, security boundaries, codified policies. Без latency overhead. | Behavior certificates ≈ evidence envelope. Но A2AS enforce на каждом action, не только на merge. |
| **Agent runtime** | **LangGraph middleware** | Before/after hooks на tool calls. Deterministic guardrails (regex, rules) + model-based. Production-ready. | Наши phase hooks — зачаток. Но они optional и не в CI path. |
| **GitHub** | **Branch protection + rulesets** | Required status checks, required workflows, merge queue. Bypass impossible для non-admins. | Не настроено. Простейший шаг, который мы не сделали. |

---

## Гипотезы

### H1: Enforcement должен быть на merge boundary, а не внутри pipeline

**Аналог:** SLSA, Kyverno (admission controller), branch protection.

**Тезис:** Единственная точка, через которую проходят ВСЕ изменения — merge в master. Контроль должен быть там, как admission controller в K8s: ресурс не создаётся, если не прошёл policy.

**Что проверить:**
- Добавить CI job `evidence-gate` + branch protection rule
- Измерить: сколько PR проходят без evidence vs с evidence
- Тест: попытаться merge PR без evidence → должен быть заблокирован

**Ресурсы:**
- GitHub branch protection: required status checks
- OPA для декларативных политик в CI
- SLSA Level 2: hosted build + provenance

### H2: Evidence должен генерироваться автоматически, а не агентом

**Аналог:** Tekton Chains (автоматическая attestation), Sigstore (keyless signing в CI).

**Тезис:** Если агент сам создаёт evidence — он может забыть или обойти. Evidence должен генерироваться автоматически из наблюдаемых артефактов (git diff, test results, CI output), а не из инструкций агенту.

**Что проверить:**
- CI job, который автоматически собирает: changed files, test coverage, lint results, beads linkage
- Сравнить с тем, что агент написал в evidence → расхождения = audit finding
- Тест: агент коммитит без evidence → CI автоматически генерирует baseline evidence

**Ресурсы:**
- Tekton Chains architecture: observer → attestation → sign → store
- in-toto: `in-toto-run` wraps commands and auto-captures materials/products
- SLSA Build L2: build platform generates provenance, not the builder

### H3: Policies должны быть декларативными и исполняемыми, а не markdown

**Аналог:** OPA/Rego, Kyverno policies, AgentSpec DSL.

**Тезис:** `quality-gates.md` и `AGENTS.md` описывают правила в markdown. Markdown не исполняется. Политики должны быть в формате, который CI/runtime может eval + enforce.

**Что проверить:**
- Перевести quality-gates в OPA/Rego или JSON Schema
- CI eval policy → pass/fail
- Тест: добавить новую policy rule → автоматически применяется без изменения кода

**Ресурсы:**
- OPA: `opa eval --data policy.rego --input pr.json --fail`
- AgentSpec: trigger → predicate → enforcement (DSL)
- Kyverno: YAML policies evaluated at admission time

### H4: Runtime enforcement нужен на каждом action, а не только на merge

**Аналог:** MI9 (FSM-conformance), AgentSpec (runtime constraints), LangGraph middleware.

**Тезис:** Merge-time enforcement ловит проблемы поздно. MI9 показывает, что FSM-conformance engine может проверять каждое действие агента в реальном времени. AgentSpec — trigger/predicate/enforcement на каждый tool call.

**Что проверить:**
- Обернуть `git commit` / `git push` в wrapper, который проверяет: есть ли beads issue, есть ли ссылка в commit message
- `sdp orchestrate` как FSM-conformance engine: каждый phase transition проверяется
- Тест: агент пытается commit без beads reference → wrapper блокирует

**Ресурсы:**
- MI9: FSM-conformance engine + goal-conditioned drift detection
- AgentSpec: runtime constraints with millisecond overhead
- A2AS: behavior certificates на каждом action

### H5: Client-side hooks бесполезны; нужен server-side enforcement

**Аналог:** Git pre-receive hooks, GitHub Actions required checks.

**Тезис:** Все наши hooks (pre-push, Stop hook) — client-side. `--no-verify` обходит pre-push. Агент может не использовать IDE hook. Server-side (CI required checks, pre-receive hooks) нельзя обойти.

**Что проверить:**
- Настроить branch protection с required checks
- Попытаться merge без прохождения checks → должно быть невозможно
- Тест: `git push --no-verify` → pre-push обходится, но CI всё равно блокирует

**Ресурсы:**
- GitHub Enterprise: pre-receive hooks (bypass impossible)
- GitHub.com: branch protection rulesets с "do not allow bypassing"
- Stack Overflow: «pre-receive hooks cannot be bypassed by regular users»

---

## Уровни enforcement (от простого к полному)

```
Level 0: Advisory (текущее состояние)
├── Правила в markdown
├── Client-side hooks (обходимы)
└── Инструменты, которые агент "должен" вызвать

Level 1: Merge Gate (минимальный enforcement)
├── CI job: evidence-gate (sdp-evidence validate)
├── Branch protection: required status checks
└── Evidence проверяется при merge — невозможно обойти

Level 2: Auto-Evidence (Tekton Chains модель)
├── CI автоматически генерирует evidence из артефактов
├── Агент НЕ МОЖЕТ забыть — evidence создаётся автоматически
├── Подпись evidence в CI (Sigstore/cosign)
└── Discrepancy между agent-evidence и auto-evidence = alert

Level 3: Declarative Policies (OPA модель)
├── Policies в Rego/JSON, не в markdown
├── CI eval policies при каждом PR
├── Новые policies применяются без изменения кода
└── Policy audit trail (кто добавил, когда, почему)

Level 4: Runtime Conformance (MI9/AgentSpec модель)
├── FSM-conformance engine проверяет каждый phase transition
├── Runtime constraints на tool calls (AgentSpec DSL)
├── Drift detection: агент отклоняется от плана → containment
└── Graduated response: warn → block → halt → escalate
```

---

## Plan для тестирования

### Immediate (1-2 сессии): Level 1

| # | Что | Как тестируем |
|---|-----|---------------|
| 1 | CI `evidence-gate` job | PR без evidence → CI fail; PR с valid evidence → CI pass |
| 2 | Branch protection | Merge без passing CI → blocked; merge с passing CI → allowed |
| 3 | Pre-push hook | `git push` с invalid evidence → blocked; `git push --no-verify` → пропускает (known limitation) |

### Short-term (3-5 сессий): Level 2

| # | Что | Как тестируем |
|---|-----|---------------|
| 4 | Auto-evidence generation | CI job auto-generates evidence from git diff + test results → compare with agent evidence |
| 5 | Evidence signing | Evidence signed in CI → verify signature at merge; unsigned evidence → rejected |
| 6 | Discrepancy detection | Agent claims 5 files changed; auto-evidence shows 8 → audit finding |

### Medium-term (5-10 сессий): Level 3

| # | Что | Как тестируем |
|---|-----|---------------|
| 7 | Policy-as-code | Rego policies for: coverage >= 80%, evidence present, beads linked → eval in CI |
| 8 | New policy auto-apply | Add policy rule → next PR automatically checked against it |
| 9 | Policy audit | Who added what policy, when, diff in git → full trail |

### Long-term: Level 4

| # | Что | Как тестируем |
|---|-----|---------------|
| 10 | FSM conformance | `sdp orchestrate` validates every transition; skip → blocked |
| 11 | AgentSpec constraints | DSL rules for tool calls; unsafe action → runtime block |
| 12 | Drift detection | Agent deviates from declared scope mid-execution → real-time alert |

---

## Key References

### Papers
- **SLSA** — https://slsa.dev/ — Supply-chain Levels for Software Artifacts
- **in-toto** — https://in-toto.io/ — Software supply chain integrity framework
- **MI9** — arXiv 2508.03858 — Runtime governance for agentic AI (FSM conformance, drift detection)
- **AgentSpec** — arXiv 2503.18666 — Runtime enforcement DSL for LLM agents (ICSE 2026)
- **A2AS** — arXiv 2510.13825 — Runtime security for AI agents ("HTTPS for AI")

### Tools
- **Tekton Chains** — https://tekton.dev/docs/chains/ — Auto-attestation for CI/CD
- **Sigstore / cosign** — https://docs.sigstore.dev/ — Keyless signing in CI
- **OPA** — https://www.openpolicyagent.org/ — Policy-as-code engine
- **Kyverno** — https://kyverno.io/ — K8s admission controller (policy engine)
- **GitHub branch protection** — required status checks, rulesets

### Agent Frameworks
- **LangGraph** — middleware/hooks for runtime guardrails
- **OpenAI Agents SDK** — input/output guardrails on tool calls
- **OpenHands** — stuck detector (external pattern recognition)

---

## Вывод

Мы не одиноки. Supply chain security решила проблему «как доказать, что pipeline выполнен правильно» (SLSA, in-toto, Tekton Chains). Policy-as-code решила «как enforce правила на merge path» (OPA, Kyverno). Agent governance решила «как контролировать агента в runtime» (MI9, AgentSpec).

Наш evidence envelope — это **attestation**. Наш `sdp orchestrate` — это **FSM conformance engine**. Наши quality gates — это **policies**. Но мы реализовали их как advisory documentation, а не как executable enforcement.

**Путь:** Level 0 (сейчас) → Level 1 (merge gate, 2 сессии) → Level 2 (auto-evidence, 5 сессий) → Level 3 (policy-as-code, 10 сессий) → Level 4 (runtime conformance, long-term).

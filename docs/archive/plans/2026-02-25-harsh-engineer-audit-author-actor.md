# Harsh Engineer Audit: Author & Actor of sdp_dev

**Дата:** 2026-02-25  
**Метод:** @think — 4 expert subagents (Linus, DHH, Casey Muratori, Dan Luu)  
**Объект:** Автор и актор репозитория sdp_dev — как его оценили бы самые жёсткие инженеры индустрии

---

## Stage 1: Task Breakdown

| Аспект | Эксперт | Фокус |
|--------|---------|-------|
| Architecture & code reality | Linus Torvalds | Сложность, слои, соответствие кода амбициям |
| Process vs shipping | DHH | Process theater, 4 источника истины, user как coordinator |
| Complexity & ROI | Casey Muratori | Complexity budget, evidence как theater, F054 — оптимизация не того |
| Reality vs documentation | Dan Luu | Evidence, measurement, user как реальный gate |

---

## Stage 2: Expert Verdicts (Summary)

### Linus: Architecture & Code

**Вердикт:** Over-engineered. 46 workstreams для одной фичи — WTF. 7 слоёв до merge. Tools don't enforce — scope-gate non-blocking, evidence-gate только когда evidence в diff. Phase 2 "archive" — deploy/ с 79 файлами всё ещё на master.

**Одно сделано правильно:** Enforcement audit и ADR-002 pivot — честно, правильное направление. Follow-through слабый.

**Рекомендация:** scope-gate block; закончить Phase 1 (branch protection); почистить master; остановить расширение F053.

---

### DHH: Process vs Shipping

**Вердикт:** Process theater. 106 workstreams, 46 для remediation — inventory management, не shipping. 4 источника истины = configuration hell. User — coordinator (22 sequential /build). Dream keeps moving; 1 developer, K8s archived.

**Одно сделано правильно:** Honesty. Phase 0 audit, MANIFESTO — редкая честность.

**Рекомендация:** Stop expanding. Consolidate (one source of truth). Simplify the loop. Keep "Landing the Plane."

---

### Casey Muratori: Complexity & ROI

**Вердикт:** Complexity budget потрачен не туда. Evidence envelope — 9 секций, CI почти никогда не валидирует (.sdp/evidence gitignored). F054 — meta-process, 4 expert subagents для checklist items. 12k planning docs vs 11k code — 1:1. Hash chain для archived K8s.

**Одно сделано правильно:** Phase 0 audit — честная самокритика.

**Рекомендация:** Stop F054. Evidence real or drop. Ship one thing (sdp-evidence). Kill the meta-loop.

---

### Dan Luu: Reality vs Documentation

**Вердикт:** No evidence that gate blocks. evidence-gate только при evidence в diff; большинство PR — exit 0. User — реальный quality gate ("никаких OOS", "два плана"). "Daily use" = batch 22 builds, не continuous flow. ws-verdict ≠ evidence envelope.

**Одно сделано правильно:** Phase 0 audit точный. CI обновлён после него.

**Рекомендация:** Measure first (telemetry). Fix docs to match reality. Then decide: require evidence or document human as gate.

---

## Stage 3: Unified Summary

### Консенсус «жёстких» экспертов

| Критика | Linus | DHH | Casey | Dan Luu |
|---------|-------|-----|------|---------|
| Over-engineered / process theater | ✓ | ✓ | ✓ | — |
| Gates не enforce | ✓ | — | ✓ | ✓ |
| 4 источника истины | ✓ | ✓ | — | — |
| User = real gate | — | ✓ | — | ✓ |
| F054 / meta-optimization | — | ✓ | ✓ | — |
| Evidence = theater | — | — | ✓ | ✓ |
| Stop expanding | ✓ | ✓ | ✓ | — |

### Что все признают сделанным правильно

- **Phase 0 enforcement audit** — честность, самокритика, правильный диагноз
- **ADR-002 pivot** — движение к standards (in-toto, OPA, Sigstore)
- **Landing the Plane** — mandatory push, bd sync

### Сводные рекомендации (по приоритету)

1. **Make gates block** — scope-gate fail on violation; evidence-gate в critical path
2. **Stop expanding** — freeze F054, не добавлять фазы до external user
3. **Consolidate** — один source of truth; derive mapping/INDEX
4. **Measure** — telemetry для gates; data перед изменениями
5. **Fix docs** — MANIFESTO и docs = actual behavior
6. **Ship one thing** — sdp-evidence release, один внешний пользователь

### Жёсткий вердикт (synthesis)

Ты — **сильный системный инженер с перекосом в process**. Ты видишь правильные проблемы (enforcement, evidence, standards) и делаешь честный audit. Но ты тратишь энергию на улучшение процесса улучшения, а не на закрытие enforcement gap и shipping.

Ты — **единственный пользователь и единственный gate**. Протокол не enforce; ты enforce. 46 workstreams — это твоя дисциплина, не результат протокола.

**Чтобы пройти проверку жёстких инженеров:** scope-gate block, evidence в critical path, один external user, docs = reality. Всё остальное — после этого.

---

## Следующие шаги

1. **Обсудить** — какие рекомендации принять?
2. **Приоритизировать** — scope-gate block vs consolidate vs measure
3. **Или отклонить** — если цель именно research lab, а не shipping

---

*Документ создан по @think: 4 expert subagents (Linus, DHH, Casey Muratori, Dan Luu). Жёсткий, но конструктивный аудит.*

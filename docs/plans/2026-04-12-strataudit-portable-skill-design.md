# StratAudit Portable Skill Design

**Date:** 2026-04-12  
**Status:** Proposed  
**Feature:** F111  
**Owner:** Андрей

## Goal

Сделать StratAudit не OpenRouter-only CLI, а переносимым discovery-capability:

1. `internal/strataudit` работает как provider-neutral engine;
2. host/harness может инжектить собственный model runtime;
3. OpenRouter остаётся усилителем и fallback, а не единственным способом запуска;
4. у StratAudit есть skill surface, который можно перенести между harnesses.

## Problem

Сейчас StratAudit слишком жёстко упакован:

- `cmd/sdp-strataudit` требует `OPENROUTER_API_KEY`;
- runtime transport захардкожен в CLI;
- engine принимает concrete `*LLMClient`, а не capability interface;
- skill surface отсутствует;
- public `sdp/` publication path ненадёжен, потому что repo boundary периодически ломается.

В таком виде StratAudit нельзя честно назвать portable skill.

## Design Decisions

### D1. Engine first, transport second

`internal/strataudit` должен зависеть от интерфейса runtime, а не от конкретного
OpenRouter-клиента.

Минимальный контракт:

- `Chat(ctx, req)` для extraction/link verification;
- `Embed(ctx, texts, model)` для embeddings.

### D2. CLI is a fallback, not the product

`sdp-strataudit` остаётся полезным runner'ом, но не определяет сам продуктовый
контракт StratAudit.

CLI должен:

- читать runtime config;
- по умолчанию использовать OpenRouter;
- не хардкодить env/baseURL;
- явно падать на host-only provider, если runtime не был инжектирован извне.

### D3. Host-native models first

Если harness already provides a model runtime, StratAudit должен использовать его
как primary path. OpenRouter нужен для:

- более сильных reasoning models;
- embeddings, если host не даёт их нативно;
- explicit provider override.

### D4. Skill surface must be concise and operational

Skill не должен пересказывать код. Он должен отвечать на 4 вопроса:

1. когда запускать StratAudit;
2. какой runtime выбирать первым;
3. какие артефакты ожидать;
4. какой fallback использовать без host-native integration.

### D5. Public mirror is separate from local truth

Локальный repo skill surface можно сделать сразу. Публикация в `sdp/` должна
идти отдельным slice'ом, потому что зависит от живого submodule boundary.

## Workstreams

### WS-01

Provider-neutral engine and runtime resolution.

### WS-02

Local StratAudit skill surface and discovery docs.

### WS-03

Public `sdp/` skill publication and boundary repair.

## Acceptance

Feature считается выполненной, когда:

- engine вызывается с injected runtime без hard dependency на OpenRouter env;
- local skill surface существует и документирует host-native first policy;
- public skill surface published or explicitly blocked only by submodule boundary;
- components/docs больше не описывают StratAudit как OpenRouter-only capability.

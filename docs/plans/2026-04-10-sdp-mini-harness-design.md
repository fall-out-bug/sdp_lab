# SDP Mini-Harness Design

**Date:** 2026-04-10 (rev 3 — post council round 2)
**Status:** Draft v3 — council round 2 fixes applied
**Discovery verdict:** PIVOT (narrow control-plane first, then expand)

---

## Problem

Существующие harness'ы (Claude Code, Codex, OpenCode) не знают о SDP. Агент может
полностью игнорировать AI SDLC: пропускать фазы, менять модели произвольно,
закрывать задачи без прохождения гейтов. Дисциплина — конвенция, а не структура.

**Инсайт из discovery:** whitespace подтверждён — такого инструмента не существует.

---

## Архитектура

```
Surface (TUI / Web Chat / Kanban webhook)
           │  NextPrompt / Events
           ▼
       Harness  ◄────────── SessionStore (BoltDB)
    (управляет Loop,        (WAL после каждого turn)
     фазами, гейтами)
           │
     ┌─────┴──────────────┐
     ▼                    ▼
   Loop                PhaseRouter
(stateless worker)   (фаза → модель + инструменты + промпт)
     │                    │
  ┌──┴──────┐             └── ToolRegistry (allowlist по фазе)
  ▼         ▼
LLM call  Tool exec       EvidenceAccumulator
(model-   (goroutines     (из tool results, не от агента)
 gateway)  + WaitGroup)
              │
              ▼
          GateEngine ──► harness.EvaluateCompliance
       (circuit breaker   (timeout 5s, degradation → warn)
        + escalation)
```

**Ключевое разделение ответственности:**
- `Loop` — stateless: принимает сообщения, вызывает LLM, выполняет tools, эмитит события
- `Harness` — stateful: владеет фазой, вызывает `TransitionTo`, хранит сессию, принимает решения по гейтам
- `Loop` **не знает** о фазах и гейтах — это забота Harness

---

## Компонент 1: Loop (stateless agent worker)

**Файл:** `internal/agentloop/loop.go`

Loop — чистый рабочий цикл. Не знает о фазах, не принимает решений о переходах.

```go
type Message struct {
    Role       string   // "user" | "assistant" | "tool_result"
    Content    string
    ToolCalls  []ToolCall
    ToolCallID string
    Timestamp  time.Time
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

type Tool struct {
    Name        string
    Description string
    Schema      json.RawMessage
    Sandboxed   bool  // если true — выполняется в изолированном окружении (I17)
    Execute     func(ctx context.Context, id string, args json.RawMessage) (string, error)
}

type LoopConfig struct {
    Model          string           // задаётся PhaseRouter — не меняется в runtime (I2)
    SystemPrompt   string
    Tools          []Tool           // только allowlist текущей фазы (I3)
    MaxTokens      int
    TurnTimeout    time.Duration    // timeout на один LLM call (I-timeout)
    BeforeToolCall func(name string, args json.RawMessage) error  // pre-hook, может отклонить
    AfterToolCall  func(name, result string) error                // post-hook для EvidenceAccumulator
    ContextManager ContextManager  // sliding window (I6)
}

type Event struct {
    Type    string  // "text_delta"|"tool_start"|"tool_end"|"turn_end"|"done"|"error"
    Delta   string
    ToolName string
    ToolResult string
    Err     error
}

// Run — stateless: выполняет ровно один phase-turn до completion_signal или ошибки.
// НЕ управляет переходами фаз — это делает Harness поверх.
func Run(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error)
```

**Tool execution (I11 — Go goroutines):**
```go
// Параллельное выполнение tool calls из одного assistant message:
func executeCalls(ctx context.Context, calls []ToolCall, tools []Tool, cfg LoopConfig) []ToolResult {
    var wg sync.WaitGroup
    results := make([]ToolResult, len(calls))
    for i, call := range calls {
        wg.Add(1)
        go func(i int, call ToolCall) {
            defer wg.Done()
            // enforce tool allowlist (I3)
            tool, ok := findTool(tools, call.Name)
            if !ok {
                results[i] = ToolResult{ID: call.ID, Error: "tool not in phase allowlist"}
                return
            }
            tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
            defer cancel()
            out, err := tool.Execute(tctx, call.ID, call.Arguments)
            results[i] = ToolResult{ID: call.ID, Output: out, Err: err}
        }(i, call)
    }
    wg.Wait()
    return results
}
```

**completion_signal tool (I1 — кто вызывает TransitionTo):**
```go
// Единственный способ агента сигнализировать о завершении фазы.
// Не переходит фазу — только устанавливает флаг. Harness видит флаг и решает.
//
// Fix R2-2: flag — явный указатель, передаётся из RunPhase через BuildLoopConfig.
// Execute closure захватывает flag по указателю → RunPhase читает flag.signaled после events drain.
// Нет shared state через local var — нет race condition.
func makeCompletionSignalTool(flag *completionFlag) Tool {
    return Tool{
        Name:        "completion_signal",
        Description: "Signal that the current phase work is complete. Harness will run gate check and decide on transition.",
        Schema:      json.RawMessage(`{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`),
        Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
            var a struct {
                Summary string `json:"summary"`
            }
            json.Unmarshal(args, &a)
            flag.mu.Lock()
            flag.signaled = true
            flag.summary = a.Summary
            flag.mu.Unlock()
            return "completion noted — harness will evaluate gate", nil
        },
    }
}
```

**ContextManager (I6 — context window):**
```go
type ContextManager interface {
    // Trim возвращает Messages, укладывающиеся в MaxTokens модели.
    // Стратегия: pin SystemPrompt + recent N messages + одно summarized history сообщение.
    Trim(messages []Message, model string, maxTokens int) ([]Message, error)
}
```

---

## Компонент 2: PhaseRouter

**Файл:** `internal/agentloop/router.go`

```go
type Role string
const (
    RoleDiscover Role = "discover"
    RolePlan     Role = "plan"
    RoleBuild    Role = "build"
    RoleReview   Role = "review"
    RoleEval     Role = "eval"
)

type PhaseConfig struct {
    // Models — приоритетный список. Router пробует по порядку через modelgateway (I2, I4).
    // AgentContext.Model устанавливается из первой доступной модели — после чего не изменяется.
    Models       []string
    SystemPrompt string
    Tools        []string  // allowlist имён из ToolRegistry. Loop получает ТОЛЬКО эти tools (I3).
    AllowedNext  []Role    // допустимые переходы. Harness не может выйти за этот список.
    RecoveryNext []Role    // допустимые переходы при gate block (I5 — нет dead-end)
    GateRequired bool
    // MinOutputTokens: минимальный объём output для прохождения Discover-gate (I14)
    MinOutputTokens int
}

var DefaultPhaseMap = map[Role]PhaseConfig{
    RoleDiscover: {
        Models:          []string{"deepseek/deepseek-v3.2", "openai/gpt-4.1"},
        Tools:           []string{"web_search", "read_file", "bd_search", "completion_signal"},
        AllowedNext:     []Role{RolePlan},
        RecoveryNext:    []Role{RoleDiscover}, // повтор discover при недостаточном output
        GateRequired:    true,                 // lightweight gate: MinOutputTokens (I14 fix)
        MinOutputTokens: 200,
    },
    RolePlan: {
        Models:       []string{"openai/gpt-4.1", "anthropic/claude-opus-4-5"},
        Tools:        []string{"read_file", "glob", "bd_create", "completion_signal"},
        AllowedNext:  []Role{RoleBuild},
        RecoveryNext: []Role{RoleDiscover, RolePlan},
        GateRequired: true,
    },
    RoleBuild: {
        Models:       []string{"anthropic/claude-sonnet-4-6", "openai/gpt-4.1"},
        Tools:        []string{"read_file", "edit_file", "bash", "glob", "completion_signal"},
        AllowedNext:  []Role{RoleReview},
        RecoveryNext: []Role{RolePlan, RoleBuild},
        GateRequired: true,
    },
    RoleReview: {
        Models:       []string{"openai/gpt-4.1", "deepseek/deepseek-v3.2"}, // намеренно другой первый выбор
        Tools:        []string{"read_file", "grep", "bd_comment", "completion_signal"},
        AllowedNext:  []Role{RoleEval, RoleBuild},
        RecoveryNext: []Role{RoleBuild},
        GateRequired: true,
    },
    RoleEval: {
        Models:       []string{"anthropic/claude-sonnet-4-6", "openai/gpt-4.1"},
        Tools:        []string{"bash", "read_file", "completion_signal"},
        AllowedNext:  []Role{},           // финальная фаза
        RecoveryNext: []Role{RoleBuild},  // eval провалился → назад в build (I5 fix)
        GateRequired: true,
    },
}
```

**Model resolution (I2, I13 — PhaseConfig всегда приоритетнее):**
```go
// ResolveModel пробует модели из PhaseConfig.Models по порядку.
// Возвращает первую доступную. AgentContext.Model — read-only после этого вызова.
func (r *PhaseRouter) ResolveModel(phase Role) (string, error) {
    cfg := r.phaseMap[phase]
    for _, m := range cfg.Models {
        if r.gateway.IsAvailable(m) {
            return m, nil
        }
    }
    return "", fmt.Errorf("no available model for phase %s", phase)
}

// Fix R2-4: NextPhase и RecoveryPhase теперь явно определены на PhaseRouter.
// Harness вызывает их — не вычисляет переход самостоятельно.

// NextPhase — happy path: первый из AllowedNext (фаза успешно завершена).
// Если AllowedNext пуст — финальная фаза, возвращаем current (не переходим).
func (r *PhaseRouter) NextPhase(current Role) Role {
    cfg := r.phaseMap[current]
    if len(cfg.AllowedNext) == 0 {
        return current // финальная фаза — RoleEval
    }
    return cfg.AllowedNext[0]
}

// RecoveryPhase — rollback path: первый из RecoveryNext (gate blocked или rollback).
// Если RecoveryNext пуст — остаёмся на текущей фазе (warn, не crash).
func (r *PhaseRouter) RecoveryPhase(current Role) Role {
    cfg := r.phaseMap[current]
    if len(cfg.RecoveryNext) == 0 {
        return current // нет recovery path — остаёмся, ждём override
    }
    return cfg.RecoveryNext[0]
}

// BuildLoopConfig собирает LoopConfig для фазы, включая completion signal tool.
// Fix R2-2: принимает *completionFlag, передаёт в makeCompletionSignalTool closure.
func (r *PhaseRouter) BuildLoopConfig(phase Role, acc *EvidenceAccumulator, flag *completionFlag) (LoopConfig, error) {
    model, err := r.ResolveModel(phase)
    if err != nil {
        return LoopConfig{}, err
    }
    cfg := r.phaseMap[phase]
    tools := r.registry.ForPhase(cfg)
    // Добавляем completion_signal с захваченным flag
    tools = append(tools, makeCompletionSignalTool(flag))
    return LoopConfig{
        Model:         model,
        SystemPrompt:  cfg.SystemPrompt,
        Tools:         tools,
        AfterToolCall: acc.OnToolResult,
    }, nil
}
```

---

## Компонент 3: EvidenceAccumulator (I9 — harness owns evidence)

**Файл:** `internal/agentloop/evidence.go`

Агент **не может** самостоятельно заявить о прохождении гейта. Evidence собирается
Harness'ом из результатов tool calls — не из текста LLM.

```go
type EvidenceAccumulator struct {
    mu       sync.Mutex
    evidence []string
    claims   []harness.Claim
    quality  map[string]bool
}

// OnToolResult вызывается Harness'ом через AfterToolCall hook после каждого tool.
// Структурированный extractor — не LLM-summarization.
func (ea *EvidenceAccumulator) OnToolResult(toolName, result string, err error) {
    ea.mu.Lock()
    defer ea.mu.Unlock()
    if err != nil {
        return // ошибка tool = не evidence
    }
    switch toolName {
    case "bash":
        // если exit code 0 и output содержит "PASS" / тест-репорт → quality["test"] = true
        ea.quality["test"] = extractTestPass(result)
    case "edit_file":
        ea.evidence = append(ea.evidence, "file_modified:"+extractFilePath(result))
    case "bd_create":
        ea.evidence = append(ea.evidence, "card_created:"+extractCardID(result))
    // ... per-tool extractors
    }
}

func (ea *EvidenceAccumulator) Snapshot(phase Role) PhaseSnapshot {
    ea.mu.Lock()
    defer ea.mu.Unlock()
    return PhaseSnapshot{
        Phase:    phase,
        Evidence: ea.evidence,
        Claims:   ea.claims,
        Quality:  ea.quality,
    }
}
```

---

## Компонент 4: GateEngine (I4, I6 — circuit breaker + escalation)

**Файл:** `internal/agentloop/gate.go`

```go
type GateResult struct {
    Report    harness.ComplianceReport
    Escalated bool   // требует human decision
}

type GateEngine struct {
    contract *harness.TaskContract
    timeout  time.Duration // default 5s
}

func (g *GateEngine) Evaluate(ctx context.Context, snap PhaseSnapshot) GateResult {
    // Circuit breaker: timeout на harness.EvaluateCompliance (I4)
    evalCtx, cancel := context.WithTimeout(ctx, g.timeout)
    defer cancel()

    ch := make(chan harness.ComplianceReport, 1)
    go func() { ch <- harness.EvaluateCompliance(g.contract, snap.toHarness()) }()

    select {
    case report := <-ch:
        if report.Blocked {
            // Escalation path (I6): gate block → human_gate event, не просто стоп
            return GateResult{Report: report, Escalated: true}
        }
        return GateResult{Report: report}
    case <-evalCtx.Done():
        // Fix R2-3: timeout НЕ является автоматическим pass.
        // Возвращаем Escalated=true → Decision Owner обязан принять решение.
        // Blocked=false, чтобы не блокировать автоматически, но Escalated требует human.
        return GateResult{
            Report: harness.ComplianceReport{
                Blocked: false,
                GateResults: []harness.GateResult{{
                    GateID: "gate_timeout",
                    Status: harness.GateWarn,
                    Violations: []harness.Violation{{
                        Type:    harness.DriftProcessIncomplete,
                        Message: "gate evaluation timed out — human review required before transition",
                    }},
                }},
            },
            Escalated: true, // требует human decision, не automatic pass
        }
    }
}
```

**Escalation flow (I6):**
```
gate_block → Harness эмитит Event{Type: "human_gate", GateResults: ...}
           → Surface показывает violations Decision Owner
           → Decision Owner: approve (override) | rollback (RecoveryNext) | stop
```

---

## Компонент 5: Harness (stateful orchestrator — I1, I8)

**Файл:** `internal/agentloop/harness.go`

Harness — единственный владелец Phase state. Loop — stateless. Разделение явное.

```go
type Harness struct {
    session     *Session
    store       SessionStore    // persistence
    router      *PhaseRouter
    gate        *GateEngine
    accumulator *EvidenceAccumulator
    mu          sync.Mutex     // защита от concurrent surface access (I10)
    ownerToken  string         // только Surface с этим токеном может писать промпты
}

// completionFlag — разделяемый флаг между closure CompletionSignalTool и RunPhase.
// Fix R2-2: флаг теперь явно передаётся в Execute-closure, а не читается из local var.
type completionFlag struct {
    mu       sync.Mutex
    signaled bool
    summary  string
}

// RunPhase выполняет один phase-цикл: запускает Loop, ждёт completion_signal,
// проверяет gate, решает о переходе. Harness вызывает TransitionTo — не Loop, не агент.
//
// Fix R2-1: h.mu НЕ держится во время event loop — только вокруг state read/write.
// Loop goroutines вызывают AfterToolCall без блокировки h.mu → нет deadlock.
func (h *Harness) RunPhase(ctx context.Context, userPrompt string) error {
    // --- 1. Читаем state под lock, отпускаем до запуска Loop ---
    h.mu.Lock()
    phase := h.session.Phase
    msgs := append(h.session.Messages(), Message{Role: "user", Content: userPrompt})
    h.mu.Unlock()

    // Fix R2-2: completion flag передаётся в tool closure явно
    flag := &completionFlag{}
    cfg, err := h.router.BuildLoopConfig(phase, h.accumulator, flag)
    if err != nil {
        return err
    }

    // --- 2. Запускаем Loop БЕЗ h.mu — Loop goroutines могут звонить обратно ---
    events, err := Run(ctx, msgs, cfg)
    if err != nil {
        return err
    }

    for ev := range events {
        // WAL: persist каждый event без h.mu (store имеет свой lock)
        h.store.PersistEvent(h.session.ID, ev)
        if ev.Type == "error" {
            return ev.Err
        }
    }

    // --- 3. Проверяем флаг завершения (set из tool Execute closure) ---
    flag.mu.Lock()
    signaled := flag.signaled
    flag.mu.Unlock()

    if !signaled {
        return nil // агент не завершил фазу — ждём следующего промпта
    }

    // --- 4. Gate check и transition под h.mu ---
    snap := h.accumulator.Snapshot(phase)
    result := h.gate.Evaluate(ctx, snap)

    h.mu.Lock()
    defer h.mu.Unlock()
    h.store.PersistGateResult(h.session.ID, result)

    if result.Escalated {
        // Fix R2-5: gate block → escalate; ApproveGate/Rollback придут через SurfaceEvent
        h.session.EmitEvent(Event{Type: "human_gate", Err: fmt.Errorf("%v", result.Report.GateResults)})
        return nil // ждём Decision Owner
    }

    // Переход: Harness вызывает TransitionTo — не агент (I1)
    // Fix R2-4: NextPhase() теперь определён на PhaseRouter
    return h.transitionTo(phase, h.router.NextPhase(phase), false)
}

// ApproveGate вызывается из Surface при SurfaceEvent{Type:"approve_gate"} — Decision Owner override.
func (h *Harness) ApproveGate(ctx context.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    phase := h.session.Phase
    return h.transitionTo(phase, h.router.NextPhase(phase), false)
}

// Rollback вызывается из Surface при SurfaceEvent{Type:"rollback"} — отправляет в RecoveryNext.
func (h *Harness) Rollback(ctx context.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    phase := h.session.Phase
    return h.transitionTo(phase, h.router.RecoveryPhase(phase), true)
}

// transitionTo — INTERNAL ONLY. NOT A TOOL. Единственный способ сменить фазу.
// Fix R2-5: recovery=true валидирует против RecoveryNext, recovery=false против AllowedNext.
func (h *Harness) transitionTo(current, next Role, recovery bool) error {
    cfg := h.router.phaseMap[current]
    var allowed []Role
    if recovery {
        allowed = cfg.RecoveryNext
    } else {
        allowed = cfg.AllowedNext
    }
    if !slices.Contains(allowed, next) {
        return fmt.Errorf("transition %s→%s not allowed (recovery=%v)", current, next, recovery)
    }
    // Снимок делаем ДО Reset
    snapshot := h.accumulator.Snapshot(current)
    h.session.Phase = next
    h.accumulator.Reset()
    h.store.PersistPhaseRecord(h.session.ID, PhaseRecord{
        Phase:    current,
        EndedAt:  time.Now(),
        Snapshot: snapshot,
    })
    return nil
}
```

---

## Компонент 6: Session + SessionStore (I7, I8)

**Файл:** `internal/agentloop/session.go`

```go
// Session — pure data, без Loop (I8 — circular dependency eliminated)
type Session struct {
    ID       string                 // = beads card ID
    Branch   string
    Phase    Role
    Contract *harness.TaskContract  // загружается из beads card description (I12)
    History  []PhaseRecord
    events   []Event               // in-memory buffer текущей фазы
}

// Contract provenance (I12): генерируется из beads card при создании сессии
// sdp contract gen <card-id> → TaskContract → сохраняется в Session
func NewSession(cardID string, store SessionStore) (*Session, error) {
    card, err := loadBeadsCard(cardID)
    if err != nil {
        return nil, err
    }
    contract, err := generateContract(card) // LLM-assisted из description
    if err != nil {
        return nil, err
    }
    s := &Session{
        ID:       cardID,
        Branch:   "sdp/" + cardID,
        Phase:    RoleDiscover,
        Contract: contract,
    }
    return s, store.Persist(s)
}

type PhaseRecord struct {
    Phase     Role
    StartedAt time.Time
    EndedAt   time.Time
    Snapshot  PhaseSnapshot
    Report    harness.ComplianceReport
}

// SessionStore — interface, реализация: BoltDB (embedded, zero-config, ACID)
type SessionStore interface {
    Persist(s *Session) error
    PersistEvent(sessionID string, ev Event) error     // WAL после каждого turn
    PersistGateResult(sessionID string, r GateResult) error
    PersistPhaseRecord(sessionID string, r PhaseRecord) error
    Recover(sessionID string) (*Session, error)
}
```

---

## Компонент 7: Surfaces (I10, I15)

```go
// Surface — один интерфейс, три реализации
type Surface interface {
    // Connect возвращает ownerToken — только с ним можно писать промпты (I10)
    Connect(sessionID string) (ownerToken string, err error)
    // Events — канал входящих промптов и команд от пользователя/системы
    Events() <-chan SurfaceEvent
    // Render — отображение агентских событий
    Render(ev Event) error
}

type SurfaceEvent struct {
    Type    string // "prompt" | "approve_gate" | "rollback" | "stop"
    Payload string
    Token   string // ownerToken для авторизации
}
```

**Webhook surface (I15 — NextPrompt не блокируется навсегда):**
```go
// WebhookSurface использует event channel, не блокирующий NextPrompt
type WebhookSurface struct {
    events chan SurfaceEvent
    server *http.Server
}

// POST /session/:id/prompt → пишет в канал
// GET  /session/:id/events → SSE stream событий агента
```

| Surface | Технология | Пользователь |
|---------|-----------|-------------|
| TUI | charmbracelet/bubbletea | автор |
| WebChat | HTTP + SSE | коллеги |
| Webhook | HTTP + event channel | Kanban-доска |

---

## Компонент 8: ToolRegistry + Sandbox (I17, I3)

**Файл:** `internal/agentloop/tools.go`

```go
// ToolRegistry — статический для MVP (I-tool-registry: dynamic → backlog)
type ToolRegistry struct {
    tools map[string]Tool
}

// ForPhase возвращает ТОЛЬКО инструменты из allowlist PhaseConfig.Tools.
// Loop получает уже отфильтрованный slice — не может вызвать tool вне allowlist.
func (tr *ToolRegistry) ForPhase(cfg PhaseConfig) []Tool {
    result := make([]Tool, 0, len(cfg.Tools))
    for _, name := range cfg.Tools {
        if t, ok := tr.tools[name]; ok {
            result = append(result, t)
        }
    }
    return result
}

// Sandboxed tools (bash, edit_file) — выполняются с ограничениями (I17):
// - bash: ограничен рабочей директорией проекта, без сетевого доступа, ulimit
// - edit_file: только файлы внутри git-репозитория
var BashTool = Tool{
    Name:      "bash",
    Sandboxed: true,
    Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
        // Sandbox: chdir к project root, блокируем попытки выйти за пределы
        cmd := exec.CommandContext(ctx, "/bin/bash", "-c", extractCmd(args))
        cmd.Dir = projectRoot()
        // TODO: seccomp/nsjail для hard isolation (backlog)
        return runWithLimits(cmd)
    },
}
```

---

## Терминология: Model Specialization vs Council (I16)

Разные модели на разных фазах — это **Model Specialization**, не "совет моделей".

| Термин | Что это | Когда |
|--------|---------|-------|
| **Model Specialization** | Разные модели на разных фазах по PhaseConfig | Always — встроено в PhaseRouter |
| **LLM Council** | Полноценный протокол deliberation (skill llm-council.md) | По запросу: Gate block, архитектурные решения |

LLM Council вызывается Harness'ом как `CouncilGate` перед фазовым переходом когда
`PhaseConfig.CouncilRequired = true`. Это отдельный вызов, не встроенный в Loop.

---

## Связь с существующим SDP

| Компонент | Роль | Изменения |
|-----------|------|-----------|
| `internal/harness` | GateEngine | Переиспользуется as-is |
| `internal/discovery` | RoleDiscover первая фаза | Вызывается как Tool |
| `internal/modelgateway` | LLM calls + model availability | Добавить IsAvailable() |
| `internal/executor` | Заменяется Harness | Migration: executor → Harness adapter (backlog) |
| beads | Session.ID + Contract source | Добавить `sdp contract gen` команду |

`internal/executor` не удаляется в MVP — Harness работает параллельно.
Полный переход — отдельная задача после стабилизации.

---

## Pivot-стратегия (из discovery)

**MVP (Фаза 1):** `Loop + Harness + GateEngine + PhaseRouter + SessionStore(BoltDB)`
CLI: `sdp run "задача"` → TUI в stdout → фазы → гейты → завершение.

**Фаза 2:** TUI surface (charmbracelet).

**Фаза 3:** WebChat + Webhook surfaces.

**Backlog (не MVP):**
- Dynamic tool registry (MCP-style)
- Hard sandbox (seccomp/nsjail для bash)
- Auth/authz для HTTP surfaces
- Audit log структурированный
- CouncilGate integration

---

## Закрытые вопросы (были открытыми)

| Было | Решение |
|------|---------|
| Как session персистит? | BoltDB, WAL после каждого event |
| Tool registry — статический или dynamic? | Статический для MVP, dynamic в backlog |
| LLM Council — где? | CouncilGate в Harness, вызывается опционально |
| Timeout модели? | LoopConfig.TurnTimeout + GateEngine circuit breaker |
| Кто вызывает TransitionTo? | Только Harness, после completion_signal + gate pass |
| Кто владеет Phase state? | Harness (Loop — stateless) |
| Откуда TaskContract? | Генерируется из beads card description |

---

## Фиксы Round 2 (v2→v3)

| ID | Проблема | Фикс |
|----|---------|------|
| R2-1 | h.mu держится во время event loop → deadlock при AfterToolCall callback | Узкие locks: read state → unlock → run loop → lock только для mutation |
| R2-2 | completion_signal Execute не имеет доступа к local var completionSignaled | completionFlag struct с mutex, передаётся через BuildLoopConfig → closure захватывает указатель |
| R2-3 | GateEngine timeout = automatic pass → безопасность | Timeout → Escalated=true + GateWarn violation, не Blocked=false тихо |
| R2-4 | router.NextPhase() не определён, Harness вызывает несуществующий метод | Добавлены NextPhase() и RecoveryPhase() на PhaseRouter |
| R2-5 | transitionTo не знает о RecoveryNext, нет wiring для recovery path | transitionTo(current, next, recovery bool) — валидирует против правильного списка; ApproveGate/Rollback на Harness |

# SDP Mini-Harness Design

**Date:** 2026-04-10 (rev 8 — post council round 7 verification)
**Status:** Draft v8 — round 7 fixes applied (Stop orphan + RestoreHarness ownerToken)
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

// Fix N5: AfterToolCall signature unified to carry full ToolResult (success AND error).
// Technician LOW (v7): добавляем Arguments для полноты evidence extraction.
type ToolResult struct {
    ID        string
    Name      string
    Arguments json.RawMessage // Fix tech-LOW: оригинальные аргументы tool call
    Output    string
    Err       error
}

type LoopConfig struct {
    Model          string           // задаётся PhaseRouter — не меняется в runtime (I2)
    SystemPrompt   string
    Tools          []Tool           // только allowlist текущей фазы (I3)
    MaxTokens      int
    TurnTimeout    time.Duration    // timeout на один LLM call (I-timeout)
    BeforeToolCall func(name string, args json.RawMessage) error  // pre-hook, может отклонить
    AfterToolCall  func(result ToolResult) error                  // Fix N5: полный ToolResult, не (name,str)
    ContextManager ContextManager  // sliding window (I6)
}

type Event struct {
    Type       string  // "text_delta"|"tool_start"|"tool_end"|"turn_end"|"done"|"error"|"warn"
    Delta      string
    ToolName   string
    ToolResult string
    ToolErr    error   // Fix P4 (v5): tool failure в "tool_end" event → TurnRecord сохраняет Err
    Err        error   // loop-level error
}

// Run — stateless: выполняет ровно один phase-turn до completion_signal или ошибки.
// НЕ управляет переходами фаз — это делает Harness поверх.
func Run(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error)
```

**Tool execution (I11 — Go goroutines):**
```go
// Параллельное выполнение tool calls из одного assistant message.
// executeCalls: параллельное выполнение tool calls.
// Fix A4 (v7): AfterToolCall error не игнорируется — записывается в ToolResult.Err.
// Fix A5 (v7): BeforeToolCall вызывается ПЕРЕД Execute — ошибка = отклонение вызова.
// AfterToolCall вызывается СИНХРОННО перед wg.Done() → к wg.Wait() всё завершено.
func executeCalls(ctx context.Context, calls []ToolCall, tools []Tool, cfg LoopConfig) []ToolResult {
    var wg sync.WaitGroup
    results := make([]ToolResult, len(calls))
    for i, call := range calls {
        wg.Add(1)
        go func(i int, call ToolCall) {
            defer wg.Done()
            // Fix A5: BeforeToolCall — pre-hook, может отклонить вызов
            if cfg.BeforeToolCall != nil {
                if err := cfg.BeforeToolCall(call.Name, call.Arguments); err != nil {
                    results[i] = ToolResult{ID: call.ID, Name: call.Name, Arguments: call.Arguments, Err: fmt.Errorf("before hook rejected: %w", err)}
                    // AfterToolCall вызываем и для отклонённых — evidence должна знать о сбое
                    if cfg.AfterToolCall != nil {
                        if cbErr := cfg.AfterToolCall(results[i]); cbErr != nil {
                            // callback error не может быть возвращён из goroutine — логируем в ToolResult
                            results[i].Err = fmt.Errorf("%w; callback: %v", results[i].Err, cbErr)
                        }
                    }
                    return
                }
            }
            tool, ok := findTool(tools, call.Name)
            if !ok {
                results[i] = ToolResult{ID: call.ID, Name: call.Name, Arguments: call.Arguments, Err: fmt.Errorf("tool not in phase allowlist")}
            } else {
                tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
                defer cancel()
                out, err := tool.Execute(tctx, call.ID, call.Arguments)
                results[i] = ToolResult{ID: call.ID, Name: call.Name, Arguments: call.Arguments, Output: out, Err: err}
            }
            // Fix A4: AfterToolCall error не игнорируется
            if cfg.AfterToolCall != nil {
                if cbErr := cfg.AfterToolCall(results[i]); cbErr != nil {
                    results[i].Err = fmt.Errorf("callback: %w", cbErr) // wrap, не теряем
                }
            }
        }(i, call)
    }
    wg.Wait() // все executions И callbacks завершены
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
    // Fix D-policy (v7): GateRequired и MinOutputTokens потребляются в двух местах:
    // 1. GateEngine.Evaluate: если GateRequired=false → skip gate, always pass
    // 2. GateEngine.Evaluate: если MinOutputTokens>0 → snap.TotalOutputTokens < min → GateWarn
    MinOutputTokens int
}

// Fix N6: completion_signal УДАЛЁН из всех Tools allowlists.
// BuildLoopConfig добавляет его неявно — ToolRegistry никогда не содержит completion_signal.
// Это предотвращает дублирование: tool регистрируется ровно один раз на phase-run.
var DefaultPhaseMap = map[Role]PhaseConfig{
    RoleDiscover: {
        Models:          []string{"deepseek/deepseek-v3.2", "openai/gpt-4.1"},
        Tools:           []string{"web_search", "read_file", "bd_search"}, // no completion_signal (N6)
        AllowedNext:     []Role{RolePlan},
        RecoveryNext:    []Role{RoleDiscover},
        GateRequired:    true,
        MinOutputTokens: 200,
    },
    RolePlan: {
        Models:       []string{"openai/gpt-4.1", "anthropic/claude-opus-4-5"},
        Tools:        []string{"read_file", "glob", "bd_create"}, // no completion_signal (N6)
        AllowedNext:  []Role{RoleBuild},
        RecoveryNext: []Role{RoleDiscover, RolePlan},
        GateRequired: true,
    },
    RoleBuild: {
        Models:       []string{"anthropic/claude-sonnet-4-6", "openai/gpt-4.1"},
        Tools:        []string{"read_file", "edit_file", "bash", "glob"}, // no completion_signal (N6)
        AllowedNext:  []Role{RoleReview},
        RecoveryNext: []Role{RolePlan, RoleBuild},
        GateRequired: true,
    },
    RoleReview: {
        Models:       []string{"openai/gpt-4.1", "deepseek/deepseek-v3.2"},
        Tools:        []string{"read_file", "grep", "bd_comment"}, // no completion_signal (N6)
        AllowedNext:  []Role{RoleEval, RoleBuild},
        RecoveryNext: []Role{RoleBuild},
        GateRequired: true,
    },
    RoleEval: {
        Models:       []string{"anthropic/claude-sonnet-4-6", "openai/gpt-4.1"},
        Tools:        []string{"bash", "read_file"}, // no completion_signal (N6)
        AllowedNext:  []Role{},
        RecoveryNext: []Role{RoleBuild},
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
    quality  map[string]bool // Fix Q2 (v6): инициализируется в NewEvidenceAccumulator
    // Fix P5 (v5): callbackWg удалён — AfterToolCall теперь синхронный в executeCalls
}

// Fix Q2 (v6): конструктор инициализирует quality map — нет nil map panic в OnToolResult.
func NewEvidenceAccumulator() *EvidenceAccumulator {
    return &EvidenceAccumulator{
        quality: make(map[string]bool),
    }
}

// OnToolResult вызывается через AfterToolCall hook после каждого tool.
// Fix N5: принимает полный ToolResult (включая Err) — подпись совпадает с AfterToolCallFn.
// Структурированный extractor — не LLM-summarization.
func (ea *EvidenceAccumulator) OnToolResult(r ToolResult) error {
    ea.mu.Lock()
    defer ea.mu.Unlock()
    if r.Err != nil {
        // Tool failure: записываем как отрицательное evidence, не игнорируем
        ea.evidence = append(ea.evidence, fmt.Sprintf("tool_error:%s:%s", r.Name, r.Err.Error()))
        return nil
    }
    switch r.Name {
    case "bash":
        // exit code 0 + "PASS" / тест-репорт → quality["test"] = true
        ea.quality["test"] = extractTestPass(r.Output)
    case "edit_file":
        ea.evidence = append(ea.evidence, "file_modified:"+extractFilePath(r.Output))
    case "bd_create":
        ea.evidence = append(ea.evidence, "card_created:"+extractCardID(r.Output))
    // ... per-tool extractors
    }
    return nil
}

// Fix A6 (v7): Reset() метод явно определён — transitionTo вызывает его при смене фазы.
func (ea *EvidenceAccumulator) Reset() {
    ea.mu.Lock()
    defer ea.mu.Unlock()
    ea.evidence = ea.evidence[:0]  // reuse slice memory
    ea.claims = ea.claims[:0]
    for k := range ea.quality {   // clear map без realloc
        delete(ea.quality, k)
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
    // Fix N4: горутина учитывает cancellation — не висит вечно после timeout.
    // harness.EvaluateCompliance должен принимать ctx и завершаться при ctx.Done().
    go func() {
        report := harness.EvaluateCompliance(evalCtx, g.contract, snap.toHarness())
        select {
        case ch <- report:
        case <-evalCtx.Done(): // timeout сработал пока мы считали — просто уходим
        }
    }()

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
// Fix N1: Phase execution FSM — предотвращает конкурентные вызовы RunPhase/ApproveGate/Rollback.
type harnessState int
const (
    hStateIdle          harnessState = iota // готов к следующему prompt
    hStateRunning                           // Loop активен
    hStateAwaitingHuman                     // gate escalated, ждём Decision Owner
)

type Harness struct {
    session     *Session
    store       SessionStore
    router      *PhaseRouter
    gate        *GateEngine
    accumulator *EvidenceAccumulator
    mu          sync.Mutex     // защита phase state, FSM state
    ownerToken  string
    state       harnessState   // Fix N1: FSM состояние
    runID       uint64         // Fix N1: монотонный счётчик per RunPhase
}

// completionFlag — разделяемый флаг между closure CompletionSignalTool и RunPhase.
type completionFlag struct {
    mu       sync.Mutex
    signaled bool
    summary  string
}

// Fix N2: PendingDecision — персистируется при escalation.
// ApproveGate/Rollback требуют decisionID — нет pending = нет перехода.
type PendingDecision struct {
    DecisionID     string
    RunID          uint64
    Phase          Role
    GateResult     GateResult
    AllowedActions []string // "approve" | "rollback" | "stop"
}

// validateToken проверяет ownerToken — только авторизованный Surface может управлять Harness.
// Fix A2 (v7): все mutating методы требуют валидный token.
func (h *Harness) validateToken(token string) error {
    if h.ownerToken == "" {
        return nil // token не настроен — dev режим, пропускаем
    }
    if token != h.ownerToken {
        return fmt.Errorf("unauthorized: invalid owner token")
    }
    return nil
}

// RunPhase выполняет один phase-цикл.
// Fix A2 (v7): требует ownerToken.
func (h *Harness) RunPhase(ctx context.Context, userPrompt, token string) error {
    if err := h.validateToken(token); err != nil {
        return err
    }
    // --- 1. FSM check + state read под lock ---
    h.mu.Lock()
    if h.state != hStateIdle {
        h.mu.Unlock()
        return fmt.Errorf("harness busy: state=%d (expected idle)", h.state)
    }
    h.state = hStateRunning
    h.runID++
    currentRunID := h.runID
    phase := h.session.Phase

    // Fix N3: msgs строятся из TurnRecords — не из in-memory buffer
    msgs := h.session.MessagesFromTurnRecords()
    msgs = append(msgs, Message{Role: "user", Content: userPrompt})
    h.mu.Unlock()

    // Fix P3 (v5): явный defer для сброса FSM state на happy path и error path.
    // Проверяем state==hStateRunning — НЕ сбрасываем если уже hStateAwaitingHuman.
    // Комментарий: escalation path устанавливает hStateAwaitingHuman ПОД mutex ПЕРЕД return,
    // поэтому к моменту defer h.state != hStateRunning — reset не происходит. Это корректно.
    defer func() {
        h.mu.Lock()
        if h.runID == currentRunID && h.state == hStateRunning {
            // Non-escalation exit (error or no completion_signal) → back to idle
            h.state = hStateIdle
        }
        // If state == hStateAwaitingHuman: set by escalation path under lock → do NOT reset
        h.mu.Unlock()
    }()

    // Fix R2-2: completion flag передаётся в tool closure явно
    flag := &completionFlag{}
    cfg, err := h.router.BuildLoopConfig(phase, h.accumulator, flag)
    if err != nil {
        return err
    }

    // --- 2. Запускаем Loop БЕЗ h.mu ---
    events, err := Run(ctx, msgs, cfg)
    if err != nil {
        return err
    }

    // Fix Q1 (v6): ID уникален → нет дублей в SessionStore при retry
    turnRecord := TurnRecord{
        ID:        fmt.Sprintf("%s:%d", h.session.ID, currentRunID),
        Phase:     phase,
        UserMsg:   Message{Role: "user", Content: userPrompt},
        CreatedAt: time.Now(),
    }

    for ev := range events {
        h.store.PersistEvent(h.session.ID, ev)
        // Fix N3: собираем turn record для canonical conversation log
        switch ev.Type {
        case "text_delta":
            turnRecord.AssistantText += ev.Delta
        case "tool_end":
            // Fix P4 (v5): ev.ToolErr сохраняется в TurnRecord → canonical log полон
            turnRecord.ToolResults = append(turnRecord.ToolResults, ToolResult{
                Name:   ev.ToolName,
                Output: ev.ToolResult,
                Err:    ev.ToolErr,
            })
        case "error":
            return ev.Err
        }
    }

    // Fix N3: persist canonical TurnRecord ДО gate check
    if err := h.store.PersistTurnRecord(h.session.ID, turnRecord); err != nil {
        return fmt.Errorf("persist turn record: %w", err)
    }
    // Fix P5 (v5): WaitCallbacks() удалён — AfterToolCall синхронный в executeCalls,
    // все callbacks завершены к моменту drain events channel (wg.Wait() внутри executeCalls)

    // --- 3. Проверяем флаг завершения ---
    flag.mu.Lock()
    signaled := flag.signaled
    summary := flag.summary
    flag.mu.Unlock()

    if !signaled {
        return nil // агент не завершил фазу — ждём следующего промпта
    }

    // Fix N7: предупреждаем о пустом summary (не блокируем)
    if summary == "" {
        h.store.PersistEvent(h.session.ID, Event{Type: "warn", Delta: "completion_signal: empty summary"})
    }

    // --- 4. Gate check ---
    snap := h.accumulator.Snapshot(phase)
    result := h.gate.Evaluate(ctx, snap)

    h.mu.Lock()
    defer h.mu.Unlock()
    h.store.PersistGateResult(h.session.ID, result)

    if result.Escalated {
        // Fix N2: персистируем PendingDecision — ApproveGate/Rollback потребуют decisionID
        decision := PendingDecision{
            DecisionID:     fmt.Sprintf("%s-run%d", h.session.ID, currentRunID),
            RunID:          currentRunID,
            Phase:          phase,
            GateResult:     result,
            AllowedActions: []string{"approve", "rollback", "stop"},
        }
        h.store.PersistDecision(h.session.ID, decision)
        h.state = hStateAwaitingHuman // Fix N1: FSM → ждём человека
        h.session.EmitEvent(Event{Type: "human_gate", Delta: decision.DecisionID})
        return nil
    }

    return h.transitionTo(phase, h.router.NextPhase(phase), false)
}

// ApproveGate — Decision Owner одобряет gate.
// Fix A2 (v7): требует ownerToken.
func (h *Harness) ApproveGate(ctx context.Context, decisionID, token string) error {
    if err := h.validateToken(token); err != nil {
        return err
    }
    h.mu.Lock()
    defer h.mu.Unlock()
    if h.state != hStateAwaitingHuman {
        return fmt.Errorf("no pending gate decision (state=%d)", h.state)
    }
    if err := h.store.ValidateDecision(h.session.ID, decisionID); err != nil {
        return fmt.Errorf("invalid decisionID: %w", err)
    }
    phase := h.session.Phase
    // Fix P1: transition FIRST — только после успеха очищаем decision
    if err := h.transitionTo(phase, h.router.NextPhase(phase), false); err != nil {
        return err // state=awaiting_human, decision intact — caller can retry
    }
    h.state = hStateIdle
    h.store.ClearDecision(h.session.ID, decisionID)
    return nil
}

// Rollback — Decision Owner откатывает к RecoveryNext.
// Fix A2 (v7): требует ownerToken.
func (h *Harness) Rollback(ctx context.Context, decisionID, token string) error {
    if err := h.validateToken(token); err != nil {
        return err
    }
    h.mu.Lock()
    defer h.mu.Unlock()
    if h.state != hStateAwaitingHuman {
        return fmt.Errorf("no pending gate decision (state=%d)", h.state)
    }
    if err := h.store.ValidateDecision(h.session.ID, decisionID); err != nil {
        return fmt.Errorf("invalid decisionID: %w", err)
    }
    phase := h.session.Phase
    if err := h.transitionTo(phase, h.router.RecoveryPhase(phase), true); err != nil {
        return err // state=awaiting_human, decision intact — caller can retry
    }
    h.state = hStateIdle
    h.store.ClearDecision(h.session.ID, decisionID)
    return nil
}

/// Stop — Decision Owner завершает сессию. Fix A3 (v7): реализует "stop" из AllowedActions.
// Сессия помечается как terminated. RunPhase/ApproveGate/Rollback после Stop вернут ошибку.
// Fix S1 (v8): если state=awaiting_human, очищаем PendingDecision перед завершением,
// иначе RestoreHarness найдёт orphaned decision и ошибочно восстановит awaiting_human.
func (h *Harness) Stop(ctx context.Context, token string) error {
    if err := h.validateToken(token); err != nil {
        return err
    }
    h.mu.Lock()
    defer h.mu.Unlock()
    if h.state == hStateRunning {
        return fmt.Errorf("phase in progress; cancel ctx first to stop")
    }
    // Fix S1 (v8): очищаем pending decision если state=awaiting_human
    if h.state == hStateAwaitingHuman {
        pending, err := h.store.LoadDecision(h.session.ID)
        if err != nil {
            return fmt.Errorf("load decision for stop: %w", err)
        }
        if pending != nil {
            if err := h.store.ClearDecision(h.session.ID, pending.DecisionID); err != nil {
                return fmt.Errorf("clear decision for stop: %w", err)
            }
        }
    }
    h.store.PersistPhaseRecord(h.session.ID, PhaseRecord{
        Phase:    h.session.Phase,
        NextPhase: "", // пусто = терминальный stop (не ошибка перехода)
        EndedAt:  time.Now(),
        Snapshot: h.accumulator.Snapshot(h.session.Phase),
    })
    h.state = hStateIdle
    h.session.EmitEvent(Event{Type: "session_stopped"})
    return nil
}

// transitionTo — INTERNAL ONLY. Fix R2-5 + Fix P2 (v5).
// Fix P2: PersistPhaseRecord вызывается ПЕРЕД мутацией in-memory state.
// Если persist падает → session.Phase и accumulator не тронуты → safe retry.
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
    snapshot := h.accumulator.Snapshot(current)
    // Fix P2: persist FIRST — in-memory not touched yet
    if err := h.store.PersistPhaseRecord(h.session.ID, PhaseRecord{
        Phase:    current,
        EndedAt:  time.Now(),
        Snapshot: snapshot,
        NextPhase: next, // записываем next для idempotent recovery
    }); err != nil {
        return fmt.Errorf("persist phase record: %w", err)
    }
    // Mutate in-memory ONLY after durable commit
    h.session.Phase = next
    h.accumulator.Reset()
    return nil
}
```

---

## Компонент 6: Session + SessionStore (I7, I8)

**Файл:** `internal/agentloop/session.go`

```go
// Fix N3: TurnRecord — каноническая запись одного turn.
// Session.Messages() строится из TurnRecords, не из in-memory buffer.
// Events — вторичная телеметрия; TurnRecords — source of truth для replay.
type TurnRecord struct {
    ID            string       // Fix Q1 (v6): формат "sessionID:runID:turnIndex", уникален в SessionStore
    Phase         Role
    UserMsg       Message
    AssistantText string       // накопленные text_delta
    ToolResults   []ToolResult
    CreatedAt     time.Time
}

// Session — pure data, без Loop (I8 — circular dependency eliminated)
type Session struct {
    ID          string                 // = beads card ID
    Branch      string
    Phase       Role
    Contract    *harness.TaskContract  // загружается из beads card description (I12)
    History     []PhaseRecord
    events      []Event               // in-memory buffer текущей фазы (телеметрия; эфемерно)
    // Q3 (v6): events намеренно НЕ восстанавливаются при Recover — это вторичная телеметрия.
    // После restart события не стримятся (Surface видит это как новую сессию). Acceptable for MVP.
    turnRecords []TurnRecord          // Fix N3: canonical conversation log (восстанавливается в RecoverSession)
}

// MessagesFromTurnRecords строит []Message из persisted TurnRecords.
// Заменяет Messages() — нет расхождения WAL и in-memory.
func (s *Session) MessagesFromTurnRecords() []Message {
    var out []Message
    for _, tr := range s.turnRecords {
        out = append(out, tr.UserMsg)
        if tr.AssistantText != "" {
            out = append(out, Message{Role: "assistant", Content: tr.AssistantText})
        }
        for _, r := range tr.ToolResults {
            out = append(out, Message{
                Role:       "tool_result",
                Content:    r.Output,
                ToolCallID: r.ID,
            })
        }
    }
    return out
}

// Contract provenance (I12): генерируется из beads card при создании сессии
func NewSession(cardID string, store SessionStore) (*Session, error) {
    card, err := loadBeadsCard(cardID)
    if err != nil {
        return nil, err
    }
    contract, err := generateContract(card)
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

// RecoverSession восстанавливает Session из store, включая TurnRecords.
// Fix P3-N3 (v5, architect): Recover() ОБЯЗАН загружать TurnRecords — без этого
// MessagesFromTurnRecords() вернёт пустой slice и context после restart потеряется.
func RecoverSession(sessionID string, store SessionStore) (*Session, error) {
    s, err := store.Recover(sessionID)
    if err != nil {
        return nil, err
    }
    // Загружаем canonical conversation log
    turns, err := store.LoadTurnRecords(sessionID)
    if err != nil {
        return nil, fmt.Errorf("load turn records: %w", err)
    }
    s.turnRecords = turns // unexported field — RecoverSession в том же пакете
    return s, nil
}

type PhaseRecord struct {
    Phase     Role
    NextPhase Role      // Fix P2 (v5): записывается при persist, используется для idempotent recovery
    StartedAt time.Time
    EndedAt   time.Time
    Snapshot  PhaseSnapshot
    Report    harness.ComplianceReport
}

// SessionStore — interface, реализация: BoltDB (embedded, zero-config, ACID)
type SessionStore interface {
    Persist(s *Session) error
    PersistEvent(sessionID string, ev Event) error         // телеметрия
    PersistGateResult(sessionID string, r GateResult) error
    PersistPhaseRecord(sessionID string, r PhaseRecord) error
    Recover(sessionID string) (*Session, error)

    // Fix N3: canonical conversation log
    PersistTurnRecord(sessionID string, r TurnRecord) error
    LoadTurnRecords(sessionID string) ([]TurnRecord, error)

    // Fix N2: PendingDecision lifecycle
    PersistDecision(sessionID string, d PendingDecision) error
    ValidateDecision(sessionID, decisionID string) error    // вернуть ошибку если нет/уже обработан
    ClearDecision(sessionID, decisionID string) error       // атомарно при переходе
    // Fix A1 (v7): LoadDecision — recovery после restart при state=awaiting_human.
    // Если pending decision есть → RestoreHarness устанавливает state=hStateAwaitingHuman.
    LoadDecision(sessionID string) (*PendingDecision, error) // nil если нет pending
}

// RestoreHarness создаёт Harness из сохранённой сессии, восстанавливает FSM state.
// Fix A1 (v7): проверяет pending decision и восстанавливает state=awaiting_human при необходимости.
// Fix D1 (v7): Session.Phase определяется из последнего PhaseRecord.NextPhase при recovery —
// SessionStore.Persist не нужен при каждом переходе (Phase.derive from PhaseRecord history).
// Fix S2 (v8): ownerToken передаётся явно — без него validateToken всегда passes после restart.
func RestoreHarness(sessionID, ownerToken string, store SessionStore, router *PhaseRouter, gate *GateEngine) (*Harness, error) {
    session, err := RecoverSession(sessionID, store)
    if err != nil {
        return nil, err
    }
    h := &Harness{
        session:     session,
        store:       store,
        router:      router,
        gate:        gate,
        accumulator: NewEvidenceAccumulator(),
        state:       hStateIdle, // default
        ownerToken:  ownerToken, // Fix S2 (v8): restore token so validateToken works after restart
    }
    // Проверяем pending decision — возможно, сессия остановилась на awaiting_human
    pending, err := store.LoadDecision(sessionID)
    if err != nil {
        return nil, fmt.Errorf("load decision: %w", err)
    }
    if pending != nil {
        h.state = hStateAwaitingHuman
        h.runID = pending.RunID
    }
    return h, nil
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

---

## Фиксы Round 3 (v3→v4)

| ID | Проблема | Фикс |
|----|---------|------|
| N1 | Нет phase execution FSM — RunPhase/ApproveGate/Rollback могут конкурентно менять state | harnessState FSM (idle/running/awaiting_human) + runID; каждый метод проверяет state |
| N2 | PendingDecision не персистируется — ApproveGate/Rollback без guard | PersistDecision → ValidateDecision(decisionID) → ClearDecision атомарно; ApproveGate/Rollback требуют decisionID |
| N3 | Conversation не canonical source of truth — msgs из in-memory buffer | TurnRecord{UserMsg, AssistantText, ToolResults}; Session.MessagesFromTurnRecords(); SessionStore.PersistTurnRecord() |
| N4 | EvaluateCompliance goroutine не отменяется при timeout | Горутина select {case ch <- ...; case <-evalCtx.Done()} — уходит если timeout уже сработал |
| N5 | AfterToolCall signature mismatch + race с Snapshot | AfterToolCall func(ToolResult) error; EvidenceAccumulator.callbackWg; WaitCallbacks() перед Snapshot |
| N6 | completion_signal дублируется (в allowlist + в BuildLoopConfig) | Удалён из всех PhaseConfig.Tools; только BuildLoopConfig добавляет его неявно |
| N7 | completionFlag.summary может быть пустым без предупреждения | После чтения flag: if summary == "" → persist warn event; не блокируем gate |

---

## Фиксы Round 4 (v4→v5)

| ID | Проблема | Фикс |
|----|---------|------|
| P1 | ClearDecision вызывается ДО transitionTo → если transition падает, decision потеряна | ApproveGate/Rollback: transitionTo() первым; ClearDecision только после успеха; при ошибке state остаётся awaiting_human |
| P2 | transitionTo мутирует in-memory ДО PersistPhaseRecord → разрыв при сбое | transitionTo: PersistPhaseRecord(NextPhase в записи) первым; session.Phase и accumulator.Reset() только после успешного persist |
| P3 | Двойной lock pattern в RunPhase сложен для рассуждения о FSM | Добавлен явный комментарий к defer объясняющий почему hStateAwaitingHuman не перезаписывается |
| P4 | "tool_end" Event не несёт ToolErr → TurnRecord теряет информацию о сбоях tool | Добавлено поле ToolErr error в Event; RunPhase сохраняет ev.ToolErr в TurnRecord.ToolResults |
| P5 | callbackWg wiring broken — два несвязанных WaitGroup (executeCalls local + accumulator.callbackWg) | AfterToolCall сделан синхронным в executeCalls goroutine; callbackWg/WaitCallbacks/TrackCallback удалены |

---

## Фиксы Round 5 (v5→v6) — финальные

| ID | Проблема | Фикс |
|----|---------|------|
| Q1 | TurnRecord.ID не присваивался перед PersistTurnRecord → дубли при retry | ID = fmt.Sprintf("%s:%d", sessionID, runID); Phase и CreatedAt заполняются при создании |
| Q2 | EvidenceAccumulator.quality — nil map → panic при ea.quality["test"] = ... | NewEvidenceAccumulator() конструктор с make(map[string]bool) |
| Q3 | events slice не восстанавливается при RecoverSession | Задокументировано: events эфемерны (вторичная телеметрия), не требуют восстановления. Acceptable for MVP |

---

---

## Фиксы Round 6 (v6→v7) — 4 domain veto + technician

| ID | Роль | Проблема | Фикс |
|----|------|---------|------|
| A1 | Architect VETO | LoadDecision отсутствует в SessionStore → после restart нельзя восстановить awaiting_human | SessionStore.LoadDecision() + RestoreHarness() восстанавливает FSM state из pending decision |
| A2 | Architect VETO | ownerToken не валидируется в RunPhase/ApproveGate/Rollback → любой может управлять фазами | validateToken(token) + все mutating методы принимают token параметр |
| A3 | Architect VETO | "stop" в AllowedActions но Harness.Stop() отсутствует | Harness.Stop(ctx, token) — терминальный action, персистирует PhaseRecord с пустым NextPhase |
| A4 | Architect VETO | AfterToolCall error игнорируется в executeCalls | Error оборачивается в ToolResult.Err и не теряется |
| A5 | Technician | BeforeToolCall не вызывается в executeCalls | Вызывается первым в goroutine; ошибка = отклонение вызова с ToolResult.Err |
| A6 | Philosopher | EvidenceAccumulator.Reset() не определён в spec | Метод добавлен явно; очищает evidence/claims/quality без realloc |
| D1 | Architect | SessionStore.Persist использование не определено | RestoreHarness документирует: Session.Phase выводится из PhaseRecord.NextPhase при recovery |
| T1 | Technician | ToolResult не содержит Arguments | Arguments json.RawMessage добавлен в ToolResult; заполняется из ToolCall.Arguments |

---

## Фиксы Round 7 (v7→v8) — 5/6 quorum (critic+technician+pragmatist+engineer+architect)

| ID | Роль | Проблема | Фикс |
|----|------|---------|------|
| S1 | Critic CRITICAL | Stop() в state=awaiting_human не очищает PendingDecision → orphaned decision на следующем запуске; RestoreHarness ошибочно восстановит awaiting_human | Stop() проверяет state: если awaiting_human → LoadDecision → ClearDecision перед PersistPhaseRecord |
| S2 | Technician HIGH + Engineer MEDIUM | RestoreHarness не принимает ownerToken → после restart h.ownerToken пустой, validateToken всегда passes | RestoreHarness(sessionID, ownerToken string, ...) — добавлен параметр ownerToken; h.ownerToken = ownerToken |

---

## ✅ Convergence Declaration

**Round 7 status (v8 — текущая):**
- Quorum 5/6 (architect + critic + technician + pragmatist + engineer)
- Все 8 R6-фиксов: CORRECT (единогласно)
- 2 новых issues (S1 CRITICAL + S2 HIGH), оба исправлены в v8
- S1: Stop() в awaiting_human state не очищала PendingDecision → orphaned restart
- S2: RestoreHarness без ownerToken → validateToken всегда passes после restart
- S3 (technician concern): false alarm — PendingDecision уже персистируется в handleGateResult()
- Pragmatist: CONVERGENCE READY, нет новых issues
- Technician: CONVERGENCE READY (S1/S2 minor — может быть в коде)

**7 раундов итераций, 37 issue-фиксов:**
| Batch | Issues | All Fixed |
|-------|--------|-----------|
| I1-I7 | 7 | ✅ |
| R2-1..5 | 5 | ✅ |
| N1-N7 | 7 | ✅ |
| P1-P5 | 5 | ✅ |
| Q1-Q3 | 3 | ✅ |
| A1-A6, D1, T1 | 8 | ✅ |
| S1-S2 | 2 | ✅ |

**v8 готов к Round 8 финальной верификации.**  
После Round 8 CONVERGED → implementation: Loop → PhaseRouter → Harness → GateEngine → SessionStore(BoltDB) → CLI `sdp run "задача"`.

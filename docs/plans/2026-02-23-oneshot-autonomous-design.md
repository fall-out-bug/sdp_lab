# @oneshot — Autonomous Execution Redesign

> **Status:** Research complete — updated with Provenance/Evidence/Drift
> **Date:** 2026-02-23
> **Goal:** Превратить @oneshot из "выполни воркстримы + сделай PR" в полностью автономный пайплайн: Feature → Branch → Drift Gate → Build → Evidence → Review Loop (0 findings) → PR → CI Loop (green) → Done

---

## Overview

### Goals

1. **Full autonomy** — от Feature ID до зелёного CI без паузы на человека
2. **Zero-findings gate** — PR создаётся только когда все P0/P1 замечания устранены
3. **CI-fix loop** — после создания PR агент мониторит CI и фиксит провалы автономно
4. **Durable execution** — context compaction не прерывает работу (checkpoint-файлы)
5. **Beads-tracked findings** — каждое замечание = beads issue с жизненным циклом
6. **Provenance chain** — полная цепочка ROADMAP → WS → Commits → Evidence → PR
7. **Drift detection gate** — блокировка выполнения если WS docs не соответствуют коду
8. **AC-level evidence** — каждый Acceptance Criteria имеет proof of satisfaction

### Key Decisions

| Aspect | Decision |
|--------|----------|
| Feature context | ROADMAP-First: читать описание/exit criteria из ROADMAP.md |
| Branch creation | Step 0 в @oneshot — idempotent, сохранять в checkpoint |
| Review-Fix loop | Two-Phase: loop(P0/P1 fix) → drain(P2/P3 в beads) → PR |
| CI-Fix loop | Step 7 inline в @oneshot: poll → classify → fix → push → repeat |
| Findings tracking | `--silent --labels "review-finding,F{NNN},round-N"` + extended verdict.json |
| Orchestration | Artifact-First State Machine через `.sdp/checkpoints/` |
| Drift detection | Pre-build gate: `sdp drift detect` per WS — HALT on ERROR, log WARNING |
| Evidence | Progressive population: intent (start) → execution → verification → trace (PR) |
| Provenance | Run file + 5 decision log points + AC-level evidence in ws-verdicts |

---

## 1. Feature Context Loading (Step 0a)

> **Expert:** Harrison Chase (LLM Orchestration), Andrew Ng, Rob Pike

### Solution

ROADMAP.md — единственный canonical source для feature-level context. Feature-level beads issues в проекте отсутствуют: `.beads-sdp-mapping.jsonl` содержит только workstream-level записи.

**Алгоритм Step 0a:**

```bash
# 1. Parse feature number
FNUM=$(echo "F067" | sed 's/F0*//')
WS_PATTERN="00-$(printf '%03d' $FNUM)-"

# 2. Verify feature exists in ROADMAP (fail fast)
if ! grep -q "F${FNUM}" docs/roadmap/ROADMAP.md; then
  echo "❌ Feature F${FNUM} not found in ROADMAP.md"
  exit 1
fi

# 3. Extract feature context
# Читать docs/roadmap/ROADMAP.md — найти строку "| F{NNN}" в Feature Index таблице
# Извлечь: feature_name, phase, size, roadmap_status, depends_on
# Найти Exit Criteria в Phase section

# 4. Check feature dependencies (через WS frontmatter, не через beads)
for dep in $feature_depends_on; do
  dep_statuses=$(grep "^status:" docs/workstreams/backlog/00-${dep}-*.md | awk '{print $2}' | sort -u)
  if echo "$dep_statuses" | grep -qv "done"; then
    echo "⚠️  WARNING: Dependency $dep has incomplete workstreams"
  fi
done

# 5. Check WS files exist
ws_files=$(ls docs/workstreams/backlog/${WS_PATTERN}*.md 2>/dev/null)
if [ -z "$ws_files" ]; then
  echo "❌ No workstream files found for F${FNUM}"; exit 1
fi

# 6. Check Beads WS statuses
bd ready --json > /tmp/bd_ready.txt
```

**Вывод агента перед стартом:**

```
╔══════════════════════════════════════════════════════╗
║  FEATURE CONTEXT: F067  <Title>                      ║
╠══════════════════════════════════════════════════════╣
║  Phase: N  │  Size: M  │  Status: Backlog            ║
║  Depends on: F066 ✅ done                            ║
╠══════════════════════════════════════════════════════╣
║  GOAL: ...                                           ║
╠══════════════════════════════════════════════════════╣
║  EXIT CRITERIA:                                      ║
║  • ...                                               ║
╠══════════════════════════════════════════════════════╣
║  WORKSTREAMS: 3 (00-067-01, 00-067-02, 00-067-03)   ║
╚══════════════════════════════════════════════════════╝
```

---

## 2. Branch Creation Strategy (Step 0b)

> **Expert:** Kelsey Hightower (DevOps/GitOps), Martin Kleppmann, Charity Majors

### Solution

**Desired state declared once, verified continuously.** @oneshot создаёт ветку один раз в Step 0b. @build только проверяет (defensive check), не создаёт.

**Алгоритм Step 0b:**

```bash
# 1. Derive branch name from ROADMAP feature title
FEATURE_TITLE=$(grep "F${FNUM}" docs/roadmap/ROADMAP.md | \
  sed 's/.*F[0-9]*[[:space:]]*//' | cut -d'|' -f1 | \
  tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | \
  sed 's/--*/-/g' | sed 's/^-//' | sed 's/-$//' | cut -c1-40)
BRANCH="feature/F${FNUM}-${FEATURE_TITLE}"

# 2. Verify clean state
if [ -n "$(git status --porcelain)" ]; then
  echo "ERROR: Uncommitted changes. Stash or commit first."; exit 1
fi

# 3. Idempotent branch setup
CURRENT=$(git branch --show-current)
if [ "$CURRENT" = "$BRANCH" ]; then
  echo "✓ Already on $BRANCH (resume mode)"
elif git show-ref --verify --quiet "refs/heads/$BRANCH"; then
  echo "✓ Branch exists, checking out: $BRANCH"
  git checkout "$BRANCH"
else
  echo "✓ Creating branch: $BRANCH"
  git fetch origin && git checkout master && git pull
  git checkout -b "$BRANCH"
fi

# 4. Persist in checkpoint (for post-compaction resume)
mkdir -p .sdp/checkpoints
# Write checkpoint — see Section 6 for full schema
```

**Defensive check в @build** (уже есть, уточнить семантику):

```bash
# @build Git Safety — VERIFY only, never create
EXPECTED=$(jq -r .branch ".sdp/checkpoints/F${FNUM}.json" 2>/dev/null)
CURRENT=$(git branch --show-current)
if [ "$CURRENT" != "$EXPECTED" ]; then
  echo "ERROR: Wrong branch. Expected $EXPECTED, got $CURRENT."
  echo "Run: git checkout $EXPECTED"
  exit 1
fi
```

---

## 3. Review-Fix Loop — Zero Findings Gate (Step 4)

> **Expert:** Shunyu Yao (ReAct/AgentOps), Andrew Ng (Reflection), Martin Fowler

### Solution

**Two-Phase approach:**
- **Phase 1** — цикл: @review → fix P0/P1 → repeat until 0 blocking или stuck
- **Phase 2** — drain: зарегистрировать P2/P3 в beads как tech debt → PR

**Критерий "0 замечаний":** `verdict == "APPROVED"` (все 6 специалистов PASS). P2/P3 findings могут существовать — они не делают специалиста FAIL по таблице @review.

**State machine:**

```
LOOP_STATE = {
  iteration: 0,
  max_iterations: 5,
  max_stalled_iterations: 2,
  previous_blocking_count: null,
  stalled_iterations: 0
}

PHASE 1: WHILE true
  LOOP_STATE.iteration++
  IF iteration > max_iterations → HALT (escalate to human)
  
  Run @review F{NNN}
  verdict = read .sdp/review_verdict.json
  
  IF verdict == "APPROVED" → break Phase 1
  
  blocking = verdict.blocking_ids (P0/P1 findings)
  
  # Stuck detection (Reflection pattern)
  IF previous_blocking_count != null:
    IF len(blocking) >= previous_blocking_count:
      stalled_iterations++
    ELSE:
      stalled_iterations = 0
    IF stalled_iterations >= max_stalled_iterations → HALT (escalate)
  previous_blocking_count = len(blocking)
  
  IF len(blocking) == 0:
    # All failures are P2/P3 — treat as APPROVED
    BREAK Phase 1
  
  FOR finding IN blocking:
    IF finding.priority == P0:
      # Inline fix in current feature branch
      sdp guard activate --finding={finding.id}
      → Fix inline
      git commit -m "fix(F{NNN}): {finding.title}"
      bd close {finding.id} --reason "fixed inline iter {N}"
    
    IF finding.priority == P1:
      # @bugfix with TDD, stay in feature branch context
      @bugfix {finding.id} --branch-from=feature/F{NNN}-xxx
      bd close {finding.id} --reason "fixed via @bugfix iter {N}"

PHASE 2: Non-blocking drain
FOR finding_id IN all P2/P3 accumulated across iterations:
  bd update {finding_id} --status=backlog --notes="Tech debt from F{NNN} review"

→ Proceed to Step 5 (Guard check → Step 6 PR)
```

**Изменения в @review** для поддержки:

1. Добавить правило в каждый субагент: `"Output PASS if ALL your findings are P2 or P3."`
2. Каждый субагент использует `--silent --labels "review-finding,F{NNN},round-{N},{role}"`
3. @review пишет в `review_verdict.json` поля `finding_ids` и `blocking_ids`

---

## 4. CI Check-Fix Loop (Step 7)

> **Expert:** Charity Majors (SRE), Martin Kleppmann, Shunyu Yao

### Solution

Добавить Step 7 inline в @oneshot после `gh pr create`. Два класса фейлов авто-фиксабельны для этого проекта (по `.github/workflows/ci.yml`): `build-test` (Go) и `k8s-validate` (Kustomize).

**Алгоритм Step 7:**

```bash
# 7.0 Capture PR
PR_NUMBER=$(gh pr list --head $(git branch --show-current) --json number -q '.[0].number')
CI_ITER=0
CI_MAX_ITER=3

# 7.1 Initial wait (CI boot ~90s)
sleep 90

# 7.2 Poll Loop
WHILE CI_ITER < CI_MAX_ITER:
  
  # OBSERVE — check CI status
  FAILING=$(gh pr checks $PR_NUMBER --json name,state -q \
    '.[] | select(.state != "SUCCESS" and .state != "SKIPPED") | .name')
  
  IF FAILING is empty:
    → CI GREEN → GOTO 7.3
  
  # REASON — classify via @ci-triage
  RUN_ID=$(gh run list --branch $(git branch --show-current) \
    --json databaseId,conclusion \
    --jq '.[] | select(.conclusion == "failure") | .databaseId' | head -1)
  gh run view $RUN_ID --log-failed > /tmp/ci-failure-$CI_ITER.log
  
  # Failure classification table:
  # Go compile/test in scope files → AUTO-FIX
  # k8s-validate                  → AUTO-FIX
  # Secrets/permissions            → ESCALATE
  # Flaky (2+ different errors)    → ESCALATE
  # Files outside feature scope    → ESCALATE
  
  IF auto-fixable:
    # ACT — targeted minimal patch
    → Read failing files from log
    → Apply minimal patch (not full @build re-run)
    → go build ./... && go test ./...   # or kubectl kustomize
    git add <failing-files>
    git commit -m "fix(CI): iter-$CI_ITER — <root cause>"
    git push origin $(git branch --show-current)
    
    # Track in beads
    bd create --title="CI-$CI_ITER: <root cause>" \
      --type=bug --priority=1 \
      --labels "ci-finding,F{NNN}"
    
    CI_ITER++
    sleep 90
    CONTINUE LOOP
  
  ELSE:
    → GOTO 7.4 (escalate)

# 7.3 CI GREEN
bd list --label ci-finding --label F{NNN} --status open \
  --json | jq -r '.[].id' | \
  xargs -I{} bd update {} --status=closed --notes="CI green on PR #$PR_NUMBER"

echo "✅ CI GREEN — @oneshot complete"
echo "PR: $PR_URL"

# 7.4 ESCALATE
bd create --title="CI BLOCKED: $FAILURE_CLASS on PR #$PR_NUMBER" \
  --type=bug --priority=0 --labels "ci-finding,F{NNN}"
echo "🚨 CI BLOCKED — manual intervention required"
```

**Принципы:**
- Initial wait 90s — CI требует ~60-90s на старт
- Poll every 30s (не exponential backoff — CI детерминирован)
- New commits, не force push — сохраняет PR history для reviewer
- Max 3 iterations с hard escalate

---

## 5. Beads as Findings Tracker

> **Expert:** Andrew Ng (Multi-agent), Martin Kleppmann, Sam Newman

### Solution

**Labels + Silent capture + Extended review_verdict.json.**

Минимальные изменения: только расширить @review (6 субагентов) для использования `--silent` и `--labels`, добавить `finding_ids`/`blocking_ids` в `review_verdict.json`.

### Label Taxonomy

| Тип finding | Labels | Кто создаёт |
|---|---|---|
| Review finding | `review-finding,F{NNN},round-{N},{role}` | @review субагент |
| CI finding | `ci-finding,F{NNN}` | @ci-triage субагент |
| WS blocker | (через frontmatter + beads) | @build / @oneshot |

### Как субагент @review создаёт finding

```bash
FINDING_ID=$(bd create \
  --title "Security: SQL injection in handler" \
  --priority 0 \
  --labels "review-finding,F067,round-1,security" \
  --type bug \
  --silent)
echo "FINDING:$FINDING_ID"
```

### Как @oneshot проверяет "0 blocking findings"

```bash
OPEN_BLOCKING=$(bd list \
  --label review-finding \
  --label F067 \
  --status open \
  --json 2>/dev/null | jq '[.[] | select(.priority <= 1)] | length')

if [ "$OPEN_BLOCKING" -eq 0 ]; then
  echo "✅ 0 blocking findings — proceed to PR"
else
  echo "❌ $OPEN_BLOCKING blocking findings remain open"
  exit 1
fi
```

### Extended review_verdict.json

```json
{
  "feature": "F067",
  "verdict": "CHANGES_REQUESTED",
  "timestamp": "...",
  "round": 1,
  "reviewers": {"qa": "PASS", "security": "FAIL"},
  "finding_ids": ["sdp_dev-abc", "sdp_dev-xyz"],
  "blocking_ids": ["sdp_dev-abc"],
  "summary": "SQL injection found in handler"
}
```

---

## 6. Skill Orchestration — Artifact-First State Machine

> **Expert:** Sam Newman (Architecture), Harrison Chase, Martin Kleppmann

### Solution

**Write-checkpoint → Execute → Verify-artifact → Advance-checkpoint** для каждого шага. Единственный checkpoint файл = полное состояние оркестрации. Любой skill возвращает результат через файловый артефакт, не через текстовый вывод.

### Checkpoint File Schema

`.sdp/checkpoints/F067.json`:

```json
{
  "schema": "1.0",
  "feature_id": "F067",
  "branch": "feature/F067-my-feature",
  "created_at": "2026-02-23T12:00:00Z",
  "updated_at": "2026-02-23T12:05:00Z",
  "phase": "build",
  "workstreams": [
    {
      "id": "00-067-01",
      "status": "done",
      "verdict_file": ".sdp/ws-verdicts/00-067-01.json",
      "commit": "abc123",
      "attempts": 1
    },
    {
      "id": "00-067-02",
      "status": "in_progress",
      "verdict_file": ".sdp/ws-verdicts/00-067-02.json",
      "commit": null,
      "attempts": 1
    }
  ],
  "review": {
    "iteration": 0,
    "verdict_file": ".sdp/review_verdict.json",
    "status": "pending"
  },
  "pr_number": null,
  "pr_url": null
}
```

### WS Verdict Contract (@build должен писать)

`.sdp/ws-verdicts/{ws-id}.json`:

```json
{
  "ws_id": "00-067-01",
  "feature_id": "F067",
  "verdict": "PASS",
  "commit": "abc123",
  "timestamp": "...",
  "quality_gates": {
    "tests": "PASS",
    "coverage": 82,
    "lint": "PASS",
    "loc_ok": true
  }
}
```

### Интерфейсный контракт каждого skill

| Skill | Input | Output artifact | Проверка |
|-------|-------|-----------------|----------|
| @build | WS ID | `.sdp/ws-verdicts/{ws-id}.json` | `jq .verdict == "PASS"` |
| @review | Feature ID | `.sdp/review_verdict.json` | `jq .verdict == "APPROVED"` |
| @beads | команды | exit code | `bd list` |
| @ci-triage | PR number | текст + beads | `gh pr checks` |

### Post-Compaction Recovery

```bash
# ПЕРВОЕ что делает @oneshot после любой compaction:
CHECKPOINT=$(ls .sdp/checkpoints/F*.json 2>/dev/null | head -1)
if [ -n "$CHECKPOINT" ]; then
  echo "=== RESUMING ==="
  cat "$CHECKPOINT"
  # Найти первый шаг не в статусе done → продолжить
else
  echo "=== NO CHECKPOINT — checking beads ==="
  bd list --status=in_progress
  bd ready
fi
```

---

## 7. Provenance, Evidence & Drift Detection

> **Expert:** Harrison Chase (LLM Orchestration / Agent chains), Martin Kleppmann, Troy Hunt, Kent C. Dodds

### Существующая инфраструктура (уже работает)

| Артефакт | Путь | Состояние |
|---|---|---|
| Evidence файлы | `.sdp/evidence/{beads_id}.json` | Схема богатая, поля пустые (`acceptance: []`, `commits: []`) |
| Run файлы | `.sdp/runs/orchestrate-{id}-{ts}.json` | Только для k8s пути, для @oneshot нет |
| Drift CLI | `sdp drift detect <ws-id>` | Работает, проверяет scope files + entities |
| Decisions CLI | `sdp decisions log/list/search` | Готов, нигде не вызывается |
| Checkpoints | `sdp checkpoint create/resume` | Работает через CLI |
| Drift skills | `@verify-workstream`, `@reality-check` | Есть, не интегрированы в @oneshot |

**Ключевой инсайт:** Инфраструктура готова. Проблема — @oneshot и @build не заполняют поля (`acceptance: []`, `commits: []`, `coverage: 0`). Нужно не строить новое, а правильно использовать существующее.

### Drift Detection Gate (Step 1.5)

Добавить между "Load Workstreams" и "Execute Workstreams":

```bash
# Step 1.5: Pre-Build Drift Gate
for ws_file in $ws_files; do
  ws_id=$(basename "$ws_file" .md)
  
  result=$(sdp drift detect "$ws_id" 2>&1)
  exit_code=$?
  
  if [ $exit_code -ne 0 ]; then
    echo "❌ DRIFT ERROR for $ws_id:"
    echo "$result"
    echo "Action: Update WS scope files OR create missing files first."
    # Append to run file
    # Log event: {phase: "drift:pre", ws_id, state: "error"}
    exit 1  # HALT
  elif echo "$result" | grep -q "WARNING"; then
    echo "⚠️  DRIFT WARNING for $ws_id: $result"
    # Log event: {phase: "drift:pre", ws_id, state: "warning", message: $result}
    # Proceed — entity might be new (created by this WS)
  else
    echo "✅ $ws_id: scope verified"
    # Log event: {phase: "drift:pre", ws_id, state: "ok"}
  fi
done
```

**Ответные действия:**

| Момент | Тип | Действие |
|--------|-----|----------|
| Step 1.5 pre-build | ERROR (scope file missing) | **HALT** — WS docs устарели или файлы нужно создать |
| Step 1.5 pre-build | WARNING (entity не найдена) | Log + proceed (entity будет создана в этом WS) |
| Step 3 post-build | ERROR (файл не создан) | Treat как @build failure → retry |
| Step 3 post-build | AC gap в ac_evidence | Create P1 finding в beads |
| Step 4 review | Documentation Expert FAIL | Standard review-fix loop |

**Известный баг:** `sdp drift detect 00-004-01` падает если WS frontmatter не имеет поля `feature` (только `feature_id`). Нужно починить в CLI или добавить `feature: F004` в WS files при Step 0a.

### Evidence File Lifecycle

**Принцип:** `intent` и `provenance` — write-once при старте WS. `execution`, `verification`, `trace` — append-only. `pr_url` — finalized в Step 6.

```
@build start (до первой строки кода):
  CREATE .sdp/evidence/{beads_id}.json
  {
    "intent": {
      "issue_id": "{beads_id}",
      "acceptance": ["{AC1}", "{AC2}"],     ← из WS frontmatter, НЕ []
      "risk_class": "medium",               ← из WS priority
      "trigger": "agent"
    },
    "plan": {
      "workstreams": ["{ws_id}"],
      "ordering_rationale": "depends_on: [...]"
    },
    "provenance": {
      "artifact_id": "{beads_id}:oneshot-evidence",
      "captured_at": "{iso_ts}",
      "contract_version": "artifact-provenance/v1",
      "orchestrator": "cursor-oneshot",
      "run_id": "oneshot-F{NNN}-{ts}",     ← из checkpoint
      "runtime": "cursor"
    }
  }

@build — execution phase:
  PATCH execution.branch = {branch}
  PATCH execution.changed_files += [changed files]

@build — after go test:
  PATCH verification.tests = ["go test ./... PASS"]
  PATCH verification.coverage.value = {actual from -coverprofile}
  PATCH verification.lint = ["go vet ./... PASS"]
  PATCH review.self_review = [
    "AC1: {text} → satisfied by {TestName} in {file}:{line}",
    "AC2: {text} → satisfied by {TestName} in {file}:{line}"
  ]

@build — after git commit:
  PATCH trace.commits = [git rev-parse HEAD]
  PATCH execution.claimed_issue_ids = ["{beads_id}"]

@oneshot Step 4 (after @review):
  PATCH review.adversarial_review = ["{summary from verdict.json}"]

@oneshot Step 6 (after gh pr create):
  FOR each evidence file in this feature:
    PATCH trace.pr_url = "{PR_URL}"
```

### AC-to-Evidence Mapping в ws-verdicts

Расширить `.sdp/ws-verdicts/{ws-id}.json` полем `ac_evidence`:

```json
{
  "ws_id": "00-067-01",
  "feature_id": "F067",
  "verdict": "PASS",
  "commit": "abc123",
  "quality_gates": {"tests": "PASS", "coverage": 82},
  "ac_evidence": [
    {
      "ac_id": "AC1",
      "ac_text": "Analyst phase creates only analyst Task",
      "evidence": "TestReconcilerAnalystOnlyPhase in agentrun_reconciler_test.go:45",
      "status": "SATISFIED"
    },
    {
      "ac_id": "AC2",
      "ac_text": "AnalystComplete reads handoff artifact",
      "evidence": "TestReconcilerAnalystCompleteReadsArtifact in agentrun_reconciler_test.go:89",
      "status": "SATISFIED"
    }
  ]
}
```

**@review Documentation Expert** верифицирует AC coverage:

```bash
WS_AC_COUNT=$(grep -c "^\- AC" docs/workstreams/backlog/{ws-id}.md || echo 0)
VERDICT_AC_COUNT=$(jq '.ac_evidence | length' .sdp/ws-verdicts/{ws-id}.json)
if [ "$WS_AC_COUNT" != "$VERDICT_AC_COUNT" ]; then
  # Create P1 finding: "AC coverage gap: N ACs in WS, M in evidence"
fi
```

### Run File (Trace @oneshot)

`.sdp/runs/oneshot-F{NNN}-{ts}.json` — создать в Step 0b, аппендить события по ходу:

```json
{
  "run_id": "oneshot-F067-20260223T120000Z",
  "feature_id": "F067",
  "orchestrator": "cursor-oneshot",
  "branch": "feature/F067-my-feature",
  "started_at": "...",
  "events": [
    {"at": "...", "phase": "init", "state": "ok", "ws_count": 3},
    {"at": "...", "phase": "drift:pre:00-067-01", "state": "ok"},
    {"at": "...", "phase": "ws:00-067-01", "state": "running"},
    {"at": "...", "phase": "ws:00-067-01", "state": "ok", "commit": "abc123"},
    {"at": "...", "phase": "drift:post:00-067-01", "state": "ok"},
    {"at": "...", "phase": "review", "state": "running", "round": 1},
    {"at": "...", "phase": "review", "state": "ok", "verdict": "APPROVED"},
    {"at": "...", "phase": "pr", "state": "ok", "pr_url": "...", "pr_number": 42},
    {"at": "...", "phase": "ci", "state": "ok", "iter": 0}
  ],
  "last_phase": "ci",
  "last_state": "ok"
}
```

Формат идентичен существующим `orchestrate-sdp_dev-*.json` в `.sdp/runs/`.

### Decision Logging — 5 обязательных точек

```bash
# 1. Feature start (Step 0a)
sdp decisions log --feature-id F{NNN} --type explicit \
  --question "Execute feature?" \
  --decision "F{NNN}: {feature_title}" \
  --rationale "ROADMAP: Phase {N}, deps {deps} ✅" \
  --maker agent

# 2. WS execution order (Step 2)
sdp decisions log --feature-id F{NNN} --type explicit \
  --question "WS execution order?" \
  --decision "Wave: {order}" \
  --rationale "Topological sort of depends_on" \
  --maker agent

# 3. Risk escalation (во время @build, если scope расширяется)
sdp decisions log --workstream-id {ws_id} --type explicit \
  --question "Risk class change?" \
  --decision "high (was {original})" \
  --rationale "Scope touched files outside declared: {files}" \
  --maker agent

# 4. Review fix strategy (Step 4, per P0/P1 finding)
sdp decisions log --feature-id F{NNN} --type explicit \
  --question "Fix strategy for {finding_id}?" \
  --decision "inline fix in {file}" \
  --rationale "P0 priority, contained in WS scope" \
  --alternatives "@bugfix separate branch" \
  --maker agent

# 5. CI escalation (Step 7, не авто-фиксабельно)
sdp decisions log --feature-id F{NNN} --type explicit \
  --question "CI failure: auto-fix possible?" \
  --decision "ESCALATE — not auto-fixable" \
  --rationale "{failure_class}: {reason}" \
  --outcome "Human intervention required" \
  --maker agent
```

### Provenance Chain — полная цепочка

```
ROADMAP.md: "F067: {title}" (Step 0a: grep → feature context)
    ↓
docs/workstreams/backlog/00-067-*.md (Step 1: WS files with AC list)
    ↓
sdp drift detect 00-067-* (Step 1.5: scope files verified)
    ↓
.sdp/evidence/{beads_id}.json: intent.acceptance populated (Step 3 start)
    ↓
git commits: execution.changed_files + trace.commits (Step 3 execution)
    ↓
go test: verification.coverage + review.self_review per-AC (Step 3 verification)
    ↓
.sdp/ws-verdicts/{ws-id}.json: ac_evidence[] (Step 3 verdict)
    ↓
.sdp/review_verdict.json: APPROVED (Step 4)
    ↓
evidence.trace.pr_url (Step 6: gh pr create)
    ↓
.sdp/runs/oneshot-F067-{ts}.json: full timeline (append throughout)
```

**Каждое звено верифицируемо без LLM:** `grep` / `jq` / `git log` / `gh pr view`.

---

## 8. Новая структура @oneshot

### Полный пайплайн

```
@oneshot F067

Step 0a: Load Feature Context
  ├── Parse F067 → FNUM=067
  ├── Verify exists in ROADMAP.md (fail fast)
  ├── Extract: feature_name, phase, exit_criteria, depends_on
  ├── Check feature dependencies (WS frontmatter statuses)
  ├── Check WS files exist
  ├── sdp decisions log: "Execute feature F067" ← DECISION POINT 1
  └── Display feature summary

Step 0b: Branch Setup
  ├── Derive branch name from feature title
  ├── Verify clean git state
  ├── Idempotent: checkout existing OR create from master
  ├── Write .sdp/checkpoints/F067.json
  └── Create .sdp/runs/oneshot-F067-{ts}.json (event: init) ← RUN FILE

Step 1: Load Workstreams
  └── Read each 00-067-*.md for: ID, depends_on, AC, scope

Step 1.5: Pre-Build Drift Gate ← NEW
  ├── For each WS: sdp drift detect {ws-id}
  │   ├── ERROR (scope file missing) → HALT, fix WS docs first
  │   └── WARNING → log to run file, proceed
  └── Log run file events: {phase: "drift:pre:{ws-id}", state}

Step 2: Build Dependency Graph
  ├── Topological sort → execution waves
  └── sdp decisions log: "WS execution order" ← DECISION POINT 2

Step 3: Execute Workstreams
  ├── For each WS in dependency order:
  │   ├── Update checkpoint: ws.status = in_progress
  │   ├── Log run file: {phase: "ws:{ws-id}", state: "running"}
  │   ├── Invoke @build (SKILL.md EXECUTE THIS NOW section)
  │   │   ├── @build creates evidence intent (acceptance from WS AC) ← EVIDENCE START
  │   │   ├── @build populates evidence execution + verification
  │   │   ├── @build writes ac_evidence[] per-AC proof ← AC MAPPING
  │   │   └── @build writes trace.commits after git commit
  │   ├── Verify .sdp/ws-verdicts/{ws-id}.json: verdict=PASS + ac_evidence filled
  │   ├── Post-build drift: sdp drift detect {ws-id} → ERROR = retry
  │   ├── Update checkpoint: ws.status = done
  │   ├── Log run file: {phase: "ws:{ws-id}", state: "ok", commit}
  │   └── bd update {beads_id} --status completed
  └── Checkpoint: phase = review

Step 4: Review-Fix Loop (Phase 1 + Phase 2)
  ├── PHASE 1: Loop until APPROVED or 0 P0/P1
  │   ├── Run @review F067
  │   │   └── Documentation Expert: sdp drift detect + AC coverage check ← DRIFT CHECK
  │   ├── Read .sdp/review_verdict.json
  │   ├── Log run file: {phase: "review", round, verdict}
  │   ├── IF APPROVED → patch evidence.review.adversarial_review → break
  │   ├── Check stuck (no reduction in blocking count)
  │   ├── Fix P0 (inline) + P1 (@bugfix with TDD)
  │   │   └── sdp decisions log: "Fix strategy" ← DECISION POINT 4 (per finding)
  │   └── bd close {finding_id} after each fix
  └── PHASE 2: Drain P2/P3 → beads as tech debt

Step 5: Verify Clean State
  ├── bd list --label review-finding --label F067 --status open → 0 blocking
  └── go test ./... → all pass

Step 6: Create PR
  ├── git push origin feature/F067-xxx
  ├── gh pr create --base dev --head feature/F067-xxx
  ├── Patch evidence.trace.pr_url for all feature WS evidence files ← EVIDENCE FINALIZE
  └── Log run file: {phase: "pr", state: "ok", pr_url, pr_number}

Step 7: CI Check-Fix Loop
  ├── Wait 90s (CI boot)
  ├── Loop (max 3 iterations):
  │   ├── Poll: gh pr checks → all green?
  │   ├── IF green → CI GREEN → update run file last_state=ok → done
  │   ├── Classify failure (auto-fixable?)
  │   ├── IF auto-fixable: patch + commit + push + wait 90s
  │   └── IF not: bd create P0 + sdp decisions log escalation + HALT ← DECISION POINT 5
  └── Close CI beads issues on green

DONE: PR URL + CI green status
      .sdp/runs/oneshot-F067-{ts}.json complete timeline
      .sdp/evidence/{id}.json files with full provenance chain
```

---

## 9. Необходимые изменения в связанных скилах

### @build (средние изменения)

1. **Создавать evidence файл при старте WS** с заполненным `intent.acceptance` из AC:
   ```bash
   # Извлечь AC из WS frontmatter
   jq -n '{"intent": {"acceptance": [...], "trigger": "agent"}, "provenance": {"run_id": "...", "orchestrator": "cursor-oneshot"}}' \
     > .sdp/evidence/${BEADS_ID}.json
   ```

2. **Записывать реальный coverage** (не 0):
   ```bash
   go test -coverprofile=/tmp/cover.out ./...
   COVERAGE=$(go tool cover -func=/tmp/cover.out | tail -1 | awk '{print $3}' | tr -d '%')
   ```

3. **Записывать verdict файл** с `ac_evidence[]` (per-AC proof):
   ```bash
   cat > .sdp/ws-verdicts/${WS_ID}.json << EOF
   {"ws_id": "${WS_ID}", "verdict": "PASS", "commit": "$(git rev-parse HEAD)",
    "ac_evidence": [{"ac_id": "AC1", "evidence": "TestXxx in file.go:NN", "status": "SATISFIED"}]}
   EOF
   ```

4. **Defensive branch check** через checkpoint (не через `sdp guard context go`):
   ```bash
   EXPECTED=$(jq -r .branch .sdp/checkpoints/F${FNUM}.json 2>/dev/null)
   CURRENT=$(git branch --show-current)
   [ -n "$EXPECTED" ] && [ "$CURRENT" != "$EXPECTED" ] && echo "ERROR: Wrong branch" && exit 1
   ```

### @review (минорные изменения)

1. **В каждый субагент промпт добавить:**
   - Rule: `Output PASS if ALL your findings are P2 or P3.`
   - Instruction: Use `bd create ... --silent --labels "review-finding,F{NNN},round-{N},{role}"` and include `FINDINGS_CREATED: {ids}` in output.

2. **Documentation Expert** — добавить AC coverage check:
   ```bash
   sdp drift detect {ws-id}
   # Проверить ac_evidence[] не пустой и покрывает все ACs
   jq '.ac_evidence | length' .sdp/ws-verdicts/{ws-id}.json
   ```

3. **After all subagents:** Агрегировать finding IDs → добавить `finding_ids` и `blocking_ids` в `review_verdict.json`.

### @verify-workstream (без изменений, вызывается по нужде)

Skill уже работает. @oneshot использует `sdp drift detect` (CLI) в Step 1.5, а `@verify-workstream` — только при интерактивном решении по дрифту ERROR.

### Новые директории

```bash
mkdir -p .sdp/checkpoints .sdp/ws-verdicts
# .sdp/evidence/ и .sdp/runs/ уже существуют
```

---

## Implementation Plan

### Phase 1: Core Redesign @oneshot

- [ ] Добавить Step 0a (Feature Context Loading) с ROADMAP parse + fail fast + `sdp decisions log`
- [ ] Добавить Step 0b (Branch Setup) с idempotent checkout + checkpoint init + run file create
- [ ] Добавить Step 1.5 (Pre-Build Drift Gate): `sdp drift detect` per WS + HALT on ERROR
- [ ] Добавить POST-COMPACTION PROTOCOL использование checkpoint файла
- [ ] Уточнить Step 3: checkpoint write/verify cycle + run file events + post-build drift check
- [ ] Переписать Step 4: Two-Phase Review-Fix Loop со stuck detection
- [ ] Добавить Step 4: `patch evidence.review.adversarial_review` после @review
- [ ] Добавить Step 5: bd verification query вместо `sdp guard finding list`
- [ ] Добавить Step 6: `patch evidence.trace.pr_url` + run file pr event
- [ ] Добавить Step 7: CI Check-Fix Loop + run file ci events

### Phase 2: Supporting Skills Updates

- [ ] **@build**: evidence file creation (intent.acceptance populated) при старте WS
- [ ] **@build**: реальный coverage measurement + запись в evidence
- [ ] **@build**: verdict file с `ac_evidence[]` per-AC mapping
- [ ] **@build**: defensive branch check через checkpoint
- [ ] **@review**: обновить субагент промпты (PASS rule + `--silent --labels` + `FINDINGS_CREATED`)
- [ ] **@review**: Documentation Expert добавить AC coverage gap check
- [ ] **@review**: агрегация `finding_ids`/`blocking_ids` в verdict файл

### Phase 3: Bug Fixes (обнаружены при анализе)

- [ ] **`sdp drift detect`**: падает на WS без поля `feature` в frontmatter (`feature_id` vs `feature`) — починить CLI или добавить `feature: F{NNN}` в WS files

### Phase 4: Validation

- [ ] Dry-run Step 0a: ROADMAP parse для F001, F004 — корректное извлечение
- [ ] Dry-run Step 1.5: `sdp drift detect` для 5 WS — нет ложных ERROR
- [ ] Full run на Feature F004 — полный пайплайн до PR
- [ ] Resume after compaction — kill session в середине Step 3, восстановление из checkpoint

---

## Success Metrics

| Metric | Baseline | Target |
|--------|----------|--------|
| Human interventions per feature | 3-5 | 0 (except escalations) |
| PR creation after review | Manual | Autonomous |
| CI green rate on first PR | ~70% | >95% (after CI-fix loop) |
| Resume after compaction | Часто теряется | 100% recovery |
| Findings left open after @oneshot | ~30% | 0% P0/P1, all P2/P3 in backlog |
| `intent.acceptance: []` in evidence | ~100% | 0% |
| `trace.commits: []` in evidence | ~100% | 0% |
| AC coverage in ws-verdicts | 0% | 100% |
| Decision log entries per feature | 0 | ≥5 |

---

## See Also

- `.cursor/skills/oneshot/SKILL.md` — текущий skill (v7.3.0)
- `.cursor/skills/review/SKILL.md` — нуждается в обновлении субагент промптов
- `.cursor/skills/build/SKILL.md` — нуждается в verdict + evidence file output
- `.cursor/skills/ci-triage/SKILL.md` — используется inline из Step 7
- `.cursor/skills/verify-workstream/SKILL.md` — вызывается при HALT на дрифте
- `.cursor/skills/protocol-consistency/SKILL.md` — аудит CLI/docs/CI consistency
- `docs/plans/2026-02-22-dream-swarm-design.md` — общий контекст архитектуры

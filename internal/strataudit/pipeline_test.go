package strataudit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/strataudit/model"
	reportpkg "github.com/fall-out-bug/sdp_lab/internal/strataudit/report"
)

var requiredAnalystTabs = []struct {
	ID    string
	Label string
}{
	{ID: "summary", Label: "Сводка"},
	{ID: "documents", Label: "Документы"},
	{ID: "trace", Label: "Трассировка"},
	{ID: "gaps", Label: "Разрывы"},
	{ID: "diagnostics", Label: "Диагностика"},
}

func TestEndToEnd_StoreAndQuery(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	// 1. Save levels
	_ = store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
		{ID: "strategy", Name: "Strategy", Rank: 1},
		{ID: "task", Name: "Task", Rank: 2},
	})

	// 2. Save documents
	_ = store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "vision.md", LevelID: "vision", ContentHash: "h1", Content: "Be the global leader"},
		{ID: "d2", Path: "strategy.md", LevelID: "strategy", ContentHash: "h2", Content: "Expand to SEA"},
		{ID: "d3", Path: "tasks.md", LevelID: "task", ContentHash: "h3", Content: "Hire country manager"},
	})

	// 3. Save entities
	_ = store.SaveEntities(ctx, []model.Entity{
		{ID: "e1", DocumentID: "d1", LevelID: "vision", Type: model.EntityGoal, Title: "Global leadership"},
		{ID: "e2", DocumentID: "d2", LevelID: "strategy", Type: model.EntityObjective, Title: "SEA expansion"},
		{ID: "e3", DocumentID: "d3", LevelID: "task", Type: model.EntityTask, Title: "Hire SG manager"},
	})

	// 4. Save traces
	_ = store.SaveTraces(ctx, []model.Trace{
		{ID: "t1", SourceEntityID: "e2", TargetEntityID: "e1", Relation: model.RelationContributesTo, Confidence: 0.9, Direction: model.DirectionUp},
		{ID: "t2", SourceEntityID: "e3", TargetEntityID: "e2", Relation: model.RelationContributesTo, Confidence: 0.85, Direction: model.DirectionUp},
	})

	// 5. Save findings
	_ = store.SaveFindings(ctx, []model.Finding{
		{ID: "f1", Type: model.FindingAlignment, Severity: model.SeverityInfo, Title: "SEA fully traced", ConfidenceScore: 0.88, LLMScore: model.LLMScoreHigh},
	})

	// Verify queries
	entities, _ := store.EntitiesByLevel(ctx, "task", model.Page{Limit: 100})
	if len(entities) != 1 {
		t.Fatalf("task entities: got %d, want 1", len(entities))
	}

	traces, _ := store.TracesForEntity(ctx, "e3")
	if len(traces) != 1 {
		t.Fatalf("traces for e3: got %d, want 1", len(traces))
	}

	findings, _ := store.FindingsByType(ctx, model.FindingAlignment, model.Page{Limit: 100})
	if len(findings) != 1 {
		t.Fatalf("alignment findings: got %d, want 1", len(findings))
	}

	// Verify coverage computation
	for _, levelID := range []string{"vision", "strategy", "task"} {
		count, _ := store.CountEntitiesByLevel(ctx, levelID)
		t.Logf("Level %s: %d entities", levelID, count)
	}

	// Verify pipeline state persistence
	_ = store.SavePipelineState(ctx, model.PipelineState{ID: "ps1", Stage: "ingest", Status: "completed", Checkpoint: `{"last": "d3"}`})
	ps, _ := store.LoadPipelineState(ctx, "ingest")
	if ps == nil || ps.Status != "completed" {
		t.Fatal("pipeline state not persisted correctly")
	}

	// Verify WAL mode
	var mode string
	_ = store.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode)
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func newMockLLMClient(t *testing.T, response string) *LLMClient {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		defer func() { _ = r.Body.Close() }()
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		call++
		payload := fmt.Sprintf(`{"choices":[{"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":4,"cost":0.0001}}`, response)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(func() {
		srv.Close()
	})
	client := NewLLMClient("test-key", srv.URL)
	client.SetRateLimit(1200)
	client.SetRetryConfig(0, 0)
	t.Cleanup(func() {
		if call == 0 {
			t.Error("mock llm was never called")
		}
	})
	return client
}

func TestExtractEntities_RejectsSourceQuoteMismatch(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	cfg := &Config{
		Levels: []LevelConfig{
			{Name: "vision", Rank: 0},
		},
		EntityTypes: []string{"goal"},
		LLM: LLMConfig{
			ExtractModel: "test-model",
		},
		Thresholds: ThresholdConfig{
			ChunkTokenLimit:    3000,
			ChunkOverlapTokens: 0,
		},
	}
	cfg.setDefaults()

	if err := store.SaveLevels(ctx, cfgToLevels(cfg)); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{
			ID:          "d1",
			Path:        "vision.md",
			LevelID:     "vision",
			ContentHash: "h1",
			Content:     "Стратегия: выйти на 30% рынка цифровых платежей в 2027.",
		},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}

	llm := newMockLLMClient(t, `{"entities":[{"type":"goal","title":"Financial target","description":"Increase share","source_quote":"This phrase does not exist in the document"}]}`)
	result, err := ExtractEntities(ctx, cfg, store, llm)
	if err != nil {
		t.Fatalf("ExtractEntities: %v", err)
	}

	entities, err := store.EntitiesByLevel(ctx, "vision", model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("EntitiesByLevel: %v", err)
	}
	if len(entities) != 0 {
		t.Fatalf("entities: got %d, want 0 (quote mismatch must be rejected as suspect path)", len(entities))
	}
	if result.EntitiesExtracted != 0 {
		t.Fatalf("result.EntitiesExtracted = %d, want 0", result.EntitiesExtracted)
	}
	if result.RejectedEntities != 1 {
		t.Fatalf("result.RejectedEntities = %d, want 1", result.RejectedEntities)
	}
}

func TestExtractEntities_RejectsBoilerplateRepetition(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	cfg := &Config{
		Levels: []LevelConfig{
			{Name: "vision", Rank: 0},
		},
		EntityTypes: []string{"goal"},
		LLM: LLMConfig{
			ExtractModel: "test-model",
		},
		Thresholds: ThresholdConfig{
			ChunkTokenLimit:    2,
			ChunkOverlapTokens: 0,
		},
	}
	cfg.setDefaults()

	if err := store.SaveLevels(ctx, cfgToLevels(cfg)); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{
			ID:          "d1",
			Path:        "vision.md",
			LevelID:     "vision",
			ContentHash: "h1",
			Content:     "This document is confidential. This document is confidential. This document is confidential.",
		},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}

	llm := newMockLLMClient(t, `{"entities":[{"type":"goal","title":"Confidentiality rule","description":"Template boilerplate","source_quote":"This document is confidential."}]}`)
	result, err := ExtractEntities(ctx, cfg, store, llm)
	if err != nil {
		t.Fatalf("ExtractEntities: %v", err)
	}

	entities, err := store.EntitiesByLevel(ctx, "vision", model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("EntitiesByLevel: %v", err)
	}
	if len(entities) != 0 {
		t.Fatalf("entities: got %d, want 0 (boilerplate repetition must not be verified truth)", len(entities))
	}
	if result.EntitiesExtracted != 0 {
		t.Fatalf("result.EntitiesExtracted = %d, want 0", result.EntitiesExtracted)
	}
	if result.RejectedEntities != 1 {
		t.Fatalf("result.RejectedEntities = %d, want 1", result.RejectedEntities)
	}
}

func TestRunPipeline_RegressionFixturePreservesTrustInvariants(t *testing.T) {
	cfg, _, result := runRegressionPipeline(t)

	if result.Extract.RejectedEntities != 1 {
		t.Fatalf("trust guarantee violated: expected exactly 1 rejected entity from prompt-leak fixture, got %d", result.Extract.RejectedEntities)
	}
	if result.Link.TracesCreated != 1 {
		t.Fatalf("trust guarantee violated: expected exactly 1 verified trace in regression fixture, got %d", result.Link.TracesCreated)
	}

	reportPath := filepath.Join(cfg.Output.Dir, "report.v2.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("trust guarantee violated: missing report.v2.json output: %v", err)
	}
	var rpt reportpkg.AuditReport
	if err := json.Unmarshal(data, &rpt); err != nil {
		t.Fatalf("trust guarantee violated: report.v2.json is not decodable: %v", err)
	}

	if rpt.SchemaVersion != reportpkg.SchemaVersion {
		t.Fatalf("trust guarantee violated: schema_version = %q, want %q", rpt.SchemaVersion, reportpkg.SchemaVersion)
	}
	if rpt.TrustSummary.Entities.Rejected != 1 {
		t.Fatalf("trust guarantee violated: rejected entity count = %d, want 1", rpt.TrustSummary.Entities.Rejected)
	}
	if len(rpt.Entities) != 2 {
		t.Fatalf("trust guarantee violated: expected 2 admitted entities after rejecting noise, got %d", len(rpt.Entities))
	}
	for _, entity := range rpt.Entities {
		if containsPromptLeakMarker(entity.Title) || containsPromptLeakMarker(entity.Description) || containsPromptLeakMarker(entity.SourceQuote) {
			t.Fatalf("trust guarantee violated: prompt/system markers leaked into admitted entity %+v", entity)
		}
		if entity.DocumentPath == "" || entity.SectionID == "" || entity.SourceQuote == "" {
			t.Fatalf("trust guarantee violated: entity lost provenance fields %+v", entity)
		}
	}

	if len(rpt.VerifiedTraces) != 1 {
		t.Fatalf("trust guarantee violated: expected 1 verified trace, got %d", len(rpt.VerifiedTraces))
	}
	trace := rpt.VerifiedTraces[0]
	if trace.TraceEvidence.Source.DocumentPath == "" || trace.TraceEvidence.Source.SectionID == "" || trace.TraceEvidence.Source.Quote == "" {
		t.Fatalf("trust guarantee violated: source trace evidence incomplete %+v", trace.TraceEvidence.Source)
	}
	if trace.TraceEvidence.Target.DocumentPath == "" || trace.TraceEvidence.Target.SectionID == "" || trace.TraceEvidence.Target.Quote == "" {
		t.Fatalf("trust guarantee violated: target trace evidence incomplete %+v", trace.TraceEvidence.Target)
	}

	if len(rpt.Coverage.ByLevel) == 0 || len(rpt.Coverage.ByDocument) == 0 || len(rpt.Coverage.BySection) == 0 {
		t.Fatalf("trust guarantee violated: coverage breakdown missing level/document/section slices %+v", rpt.Coverage)
	}
	if len(rpt.CorpusQuality.Documents) == 0 {
		t.Fatal("trust guarantee violated: corpus_quality block lost noisy document diagnostics")
	}

	compatBytes, err := os.ReadFile(filepath.Join(cfg.Output.Dir, "report.json"))
	if err != nil {
		t.Fatalf("trust guarantee violated: missing compatibility report.json alias: %v", err)
	}
	if strings.TrimSpace(string(compatBytes)) != strings.TrimSpace(string(data)) {
		t.Fatal("trust guarantee violated: report.json alias diverged from report.v2.json")
	}

	diagBytes, err := os.ReadFile(filepath.Join(cfg.Output.Dir, "llm_diagnostics.json"))
	if err != nil {
		t.Fatalf("trust guarantee violated: missing llm_diagnostics.json artifact: %v", err)
	}
	var diagnostics struct {
		Summary struct {
			Total   int            `json:"total"`
			ByStage map[string]int `json:"by_stage"`
		} `json:"summary"`
		Invocations []struct {
			Stage         string            `json:"stage"`
			ContentSource string            `json:"content_source"`
			Metadata      map[string]string `json:"metadata"`
		} `json:"invocations"`
	}
	if err := json.Unmarshal(diagBytes, &diagnostics); err != nil {
		t.Fatalf("trust guarantee violated: llm_diagnostics.json is not decodable: %v", err)
	}
	if diagnostics.Summary.Total == 0 {
		t.Fatal("trust guarantee violated: llm diagnostics summary recorded zero invocations")
	}
	if diagnostics.Summary.ByStage["extract"] == 0 {
		t.Fatalf("trust guarantee violated: llm diagnostics missing extract stage counts %+v", diagnostics.Summary.ByStage)
	}
	if len(diagnostics.Invocations) == 0 || diagnostics.Invocations[0].Metadata["document_id"] == "" {
		t.Fatalf("trust guarantee violated: llm diagnostics lost request metadata %+v", diagnostics.Invocations)
	}

	htmlBytes, err := os.ReadFile(filepath.Join(cfg.Output.Dir, "report.html"))
	if err != nil {
		t.Fatalf("trust guarantee violated: missing HTML report: %v", err)
	}
	html := string(htmlBytes)
	if !strings.Contains(html, "company-vision.md") || !strings.Contains(html, "#section-0") {
		t.Fatalf("trust guarantee violated: HTML report lost document/section provenance preview\n%s", html)
	}
	if !strings.Contains(html, "Template memo.") {
		t.Fatalf("trust guarantee violated: HTML report lost evidence preview for noisy corpus section\n%s", html)
	}
}

func TestRunPipeline_RegressionFixtureLocksTabbedAnalystUX(t *testing.T) {
	cfg, _, _ := runRegressionPipeline(t)
	rpt, html := loadRegressionFixtureReport(t, cfg.Output.Dir)

	assertRegressionReportTabs(t, rpt, html)
	assertRegressionReportStartsInSummaryMode(t, rpt, html)
	assertRegressionReportUsesRussianChrome(t, html)

	strategyView := findDocumentViewByBaseName(t, rpt.DocumentViews, "payment-strategy.md")
	if len(strategyView.UpstreamDocuments) != 1 {
		t.Fatalf("expected exactly one explainable upstream document for payment-strategy.md, got %+v", strategyView.UpstreamDocuments)
	}
	upstream := strategyView.UpstreamDocuments[0]
	if filepath.Base(upstream.DocumentPath) != "company-vision.md" {
		t.Fatalf("strategy upstream document = %q, want company-vision.md", upstream.DocumentPath)
	}
	if upstream.VerifiedEdgeCount != 1 {
		t.Fatalf("strategy upstream verified_edge_count = %d, want 1", upstream.VerifiedEdgeCount)
	}
	if len(upstream.ClaimIDs) == 0 || len(upstream.EdgeIDs) == 0 {
		t.Fatalf("strategy upstream correspondence lost explainability: %+v", upstream)
	}

	visionView := findDocumentViewByBaseName(t, rpt.DocumentViews, "company-vision.md")
	if len(visionView.DownstreamDocuments) != 1 {
		t.Fatalf("expected exactly one explainable downstream document for company-vision.md, got %+v", visionView.DownstreamDocuments)
	}
	downstream := visionView.DownstreamDocuments[0]
	if filepath.Base(downstream.DocumentPath) != "payment-strategy.md" {
		t.Fatalf("vision downstream document = %q, want payment-strategy.md", downstream.DocumentPath)
	}
	if downstream.VerifiedEdgeCount != 1 {
		t.Fatalf("vision downstream verified_edge_count = %d, want 1", downstream.VerifiedEdgeCount)
	}
	if len(downstream.ClaimIDs) == 0 || len(downstream.EdgeIDs) == 0 {
		t.Fatalf("vision downstream correspondence lost explainability: %+v", downstream)
	}
}

func TestRunPipeline_RegressionFixtureZeroTraceExplainsGapWaterfall(t *testing.T) {
	cfg, _, result := runRegressionPipelineWithRuntime(t, newRegressionZeroTraceRuntime(t))
	if result.Link.CandidatesGenerated == 0 {
		t.Fatal("zero-trace regression fixture produced no trace candidates; gap waterfall would be meaningless")
	}
	if result.Link.TracesCreated != 0 {
		t.Fatalf("zero-trace regression fixture created %d verified traces, want 0", result.Link.TracesCreated)
	}

	rpt, html := loadRegressionFixtureReport(t, cfg.Output.Dir)
	assertRegressionReportTabs(t, rpt, html)
	assertRegressionReportStartsInSummaryMode(t, rpt, html)

	if len(rpt.VerifiedTraces) != 0 {
		t.Fatalf("zero-trace regression fixture must not emit verified traces, got %d", len(rpt.VerifiedTraces))
	}
	if len(rpt.TraceGaps) == 0 {
		t.Fatal("zero-trace regression fixture lost trace_gaps entirely")
	}

	var rejectedGap *reportpkg.TraceGapReport
	for i := range rpt.TraceGaps {
		gap := &rpt.TraceGaps[i]
		if gap.Stage == "verification" && gap.GapType == "all_candidates_rejected" {
			rejectedGap = gap
			break
		}
	}
	if rejectedGap == nil {
		t.Fatalf("zero-trace regression fixture missing verification rejection gap: %+v", rpt.TraceGaps)
	}
	if rejectedGap.CandidateCount == 0 || len(rejectedGap.TopCandidateIDs) == 0 {
		t.Fatalf("zero-trace rejection gap lost candidate evidence: %+v", *rejectedGap)
	}
	if rejectedGap.Reason != "llm_verification_rejected" {
		t.Fatalf("zero-trace rejection gap reason = %q, want llm_verification_rejected", rejectedGap.Reason)
	}

	strategyView := findDocumentViewByBaseName(t, rpt.DocumentViews, "payment-strategy.md")
	if strategyView.BlockerCount == 0 || strategyView.BrokenLinkCount == 0 {
		t.Fatalf("strategy document lost blocker accounting in zero-trace mode: %+v", strategyView)
	}
	if len(strategyView.Blockers) == 0 || strategyView.Blockers[0].GapType != "all_candidates_rejected" {
		t.Fatalf("strategy document blockers do not explain zero-trace failure: %+v", strategyView.Blockers)
	}

	for _, needle := range []string{
		"Подтверждённых трасс нет",
		"Водопад разрывов",
		"Разрывы по утверждениям",
		"Открыть разрывы",
	} {
		if !strings.Contains(html, needle) {
			t.Fatalf("zero-trace HTML report missing %q\n%s", needle, html)
		}
	}
}

func loadRegressionFixtureReport(t *testing.T, outputDir string) (*reportpkg.AuditReport, string) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(outputDir, "report.v2.json"))
	if err != nil {
		t.Fatalf("ReadFile(report.v2.json): %v", err)
	}
	var rpt reportpkg.AuditReport
	if err := json.Unmarshal(data, &rpt); err != nil {
		t.Fatalf("Unmarshal(report.v2.json): %v", err)
	}

	htmlBytes, err := os.ReadFile(filepath.Join(outputDir, "report.html"))
	if err != nil {
		t.Fatalf("ReadFile(report.html): %v", err)
	}
	return &rpt, string(htmlBytes)
}

func assertRegressionReportTabs(t *testing.T, rpt *reportpkg.AuditReport, html string) {
	t.Helper()

	if len(rpt.ReportModes.Tabs) != len(requiredAnalystTabs) {
		t.Fatalf("report_modes.tabs len = %d, want %d", len(rpt.ReportModes.Tabs), len(requiredAnalystTabs))
	}
	for i, tab := range requiredAnalystTabs {
		if rpt.ReportModes.Tabs[i].ID != tab.ID || rpt.ReportModes.Tabs[i].Label != tab.Label {
			t.Fatalf("report_modes.tabs[%d] = %+v, want id=%q label=%q", i, rpt.ReportModes.Tabs[i], tab.ID, tab.Label)
		}
		if !strings.Contains(html, fmt.Sprintf(`data-tab-btn="%s"`, tab.ID)) {
			t.Fatalf("HTML report missing tab button for %q\n%s", tab.ID, html)
		}
		if !strings.Contains(html, fmt.Sprintf(`id="tab-%s"`, tab.ID)) {
			t.Fatalf("HTML report missing tab panel for %q\n%s", tab.ID, html)
		}
		if !strings.Contains(html, tab.Label) {
			t.Fatalf("HTML report missing tab label %q\n%s", tab.Label, html)
		}
	}
}

func assertRegressionReportStartsInSummaryMode(t *testing.T, rpt *reportpkg.AuditReport, html string) {
	t.Helper()

	if rpt.ReportModes.Default != "analyst" {
		t.Fatalf("report_modes.default = %q, want analyst", rpt.ReportModes.Default)
	}
	if rpt.ReportModes.DefaultTab != "summary" {
		t.Fatalf("report_modes.default_tab = %q, want summary", rpt.ReportModes.DefaultTab)
	}
	if rpt.ReportModes.CompareAvailable {
		t.Fatal("report_modes.compare_available must stay false for analyst-first flow")
	}
	if !strings.Contains(html, "Стартовая вкладка: Сводка") {
		t.Fatalf("HTML report lost summary-first cue\n%s", html)
	}
	if !strings.Contains(html, "Сравнение прогонов: выключен") {
		t.Fatalf("HTML report lost compare-off cue\n%s", html)
	}
	summaryIdx := strings.Index(html, `id="tab-summary"`)
	diagnosticsIdx := strings.Index(html, `id="tab-diagnostics"`)
	if summaryIdx == -1 || diagnosticsIdx == -1 {
		t.Fatalf("HTML report missing summary/diagnostics sections\n%s", html)
	}
	if summaryIdx > diagnosticsIdx {
		t.Fatalf("summary tab rendered after diagnostics: summary=%d diagnostics=%d", summaryIdx, diagnosticsIdx)
	}
}

func assertRegressionReportUsesRussianChrome(t *testing.T, html string) {
	t.Helper()

	for _, needle := range []string{
		"Навигация отчёта",
		"Режим: аналитический",
		"Критические документы корпуса",
		"Водопад разрывов",
	} {
		if !strings.Contains(html, needle) {
			t.Fatalf("HTML report missing Russian chrome label %q\n%s", needle, html)
		}
	}
	for _, needle := range []string{
		"Executive Overview",
		"Analyst Explorer",
		"Evidence Pack",
		"Compare mode:",
		"All findings",
	} {
		if strings.Contains(html, needle) {
			t.Fatalf("HTML report leaked English chrome %q\n%s", needle, html)
		}
	}
}

func findDocumentViewByBaseName(t *testing.T, views []reportpkg.DocumentViewReport, base string) reportpkg.DocumentViewReport {
	t.Helper()

	for _, view := range views {
		if filepath.Base(view.DocumentPath) == base {
			return view
		}
	}
	t.Fatalf("document_view for %q not found in %+v", base, views)
	return reportpkg.DocumentViewReport{}
}

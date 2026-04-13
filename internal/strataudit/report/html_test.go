package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildHTML_RendersThreeLayerEvidenceReport(t *testing.T) {
	goldenBytes, err := os.ReadFile(filepath.Join("testdata", "report_v2.golden.json"))
	if err != nil {
		t.Fatalf("ReadFile(report_v2.golden.json): %v", err)
	}

	var rpt AuditReport
	if err := json.Unmarshal(goldenBytes, &rpt); err != nil {
		t.Fatalf("Unmarshal(report_v2.golden.json): %v", err)
	}

	out := buildHTML(&rpt)
	for _, needle := range []string{
		`data-tab-btn="summary"`,
		`data-tab-btn="documents"`,
		`data-tab-btn="trace"`,
		`data-tab-btn="gaps"`,
		`data-tab-btn="diagnostics"`,
		`Сводка`,
		`Документы`,
		`Трассировка`,
		`Разрывы`,
		`Диагностика`,
		`Режим: аналитический`,
		`Сравнение прогонов: выключен`,
		`Стартовая вкладка: Сводка`,
		`Покрытие по слоям`,
		`Покрытие по документам`,
		`Покрытие по разделам`,
		`Критические документы корпуса`,
		`Водопад разрывов`,
		`href="#doc-d1"`,
		`href="#section-s1"`,
		`href="#evidence-entity-e1"`,
		`Связи вверх по слоям`,
		`Блокеры трассировки`,
		`Ключевые утверждения документа`,
		`id="trace-tr1"`,
		`id="tab-diagnostics"`,
		`file:///tmp/vision.md`,
		`Есть suspect-сущн.; часть выводов требует ручной проверки.`,
		`Evidence-backed relation.`,
		`report.v2.json`,
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("HTML output missing %q\n%s", needle, out)
		}
	}
	for _, needle := range []string{
		`Executive Overview`,
		`Analyst Explorer`,
		`Evidence Pack`,
		`Compare mode:`,
		`All findings`,
	} {
		if strings.Contains(out, needle) {
			t.Fatalf("HTML output should not contain %q\n%s", needle, out)
		}
	}
}

func TestBuildHTML_KeepsSummaryFirstRussianAnalystFlow(t *testing.T) {
	goldenBytes, err := os.ReadFile(filepath.Join("testdata", "report_v2.golden.json"))
	if err != nil {
		t.Fatalf("ReadFile(report_v2.golden.json): %v", err)
	}

	var rpt AuditReport
	if err := json.Unmarshal(goldenBytes, &rpt); err != nil {
		t.Fatalf("Unmarshal(report_v2.golden.json): %v", err)
	}

	out := buildHTML(&rpt)
	if !strings.Contains(out, "Стартовая вкладка: Сводка") {
		t.Fatalf("summary-first analyst flow cue missing\n%s", out)
	}
	if !strings.Contains(out, "Сравнение прогонов: выключен") {
		t.Fatalf("compare-off analyst flow cue missing\n%s", out)
	}
	if !strings.Contains(out, "Навигация отчёта") || !strings.Contains(out, "Режим: аналитический") {
		t.Fatalf("russian analyst chrome missing\n%s", out)
	}
	summaryIdx := strings.Index(out, `id="tab-summary"`)
	diagnosticsIdx := strings.Index(out, `id="tab-diagnostics"`)
	if summaryIdx == -1 || diagnosticsIdx == -1 {
		t.Fatalf("summary/diagnostics tabs missing\n%s", out)
	}
	if summaryIdx > diagnosticsIdx {
		t.Fatalf("summary tab rendered after diagnostics: summary=%d diagnostics=%d", summaryIdx, diagnosticsIdx)
	}
}

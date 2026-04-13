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
		`Executive Overview`,
		`Analyst Explorer`,
		`Evidence Pack`,
		`Coverage by Level`,
		`Coverage by Document`,
		`Coverage by Section`,
		`Corpus Quality Blockers`,
		`Trace Explorer`,
		`href="#doc-d1"`,
		`href="#section-s1"`,
		`href="#evidence-entity-e1"`,
		`id="trace-tr1"`,
		`file:///tmp/vision.md`,
		`Есть suspect-сущн.; часть выводов требует ручной проверки.`,
		`Evidence-backed relation.`,
		`report.v2.json`,
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("HTML output missing %q\n%s", needle, out)
		}
	}
}

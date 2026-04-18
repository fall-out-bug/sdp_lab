// Package eval provides metrics computation for evaluation harness.
// It computes precision, recall, and F1 scores per ecosystem.
package eval

import (
	"fmt"
	"strings"

	"sdp_dev/internal/architect"
)

// EcosystemMetrics holds aggregated metrics for a specific ecosystem.
type EcosystemMetrics struct {
	Ecosystem string `json:"ecosystem"`

	// Import accuracy metrics
	ImportPrecision float64 `json:"import_precision"`
	ImportRecall    float64 `json:"import_recall"`
	ImportF1        float64 `json:"import_f1"`

	// Style hypothesis metrics
	StylePrecision float64 `json:"style_precision"`
	StyleRecall    float64 `json:"style_recall"`
	StyleF1        float64 `json:"style_f1"`

	// C4 generation metrics
	C4Precision float64 `json:"c4_precision"`
	C4Recall    float64 `json:"c4_recall"`
	C4F1        float64 `json:"c4_f1"`

	// Schema accuracy (for SQL)
	SchemaPrecision float64 `json:"schema_precision,omitempty"`
	SchemaRecall    float64 `json:"schema_recall,omitempty"`
	SchemaF1        float64 `json:"schema_f1,omitempty"`

	OverallScore float64 `json:"overall_score"`
	SampleCount  int     `json:"sample_count"`
}

// MetricsAggregator aggregates metrics across multiple evaluation runs.
type MetricsAggregator struct {
	byEcosystem map[string]*ecosystemAccumulator
}

type ecosystemAccumulator struct {
	importTP, importFP, importFN int
	styleTP, styleFP, styleFN   int
	c4TP, c4FP, c4FN           int
	schemaTP, schemaFP, schemaFN int
	sampleCount                 int
}

// NewMetricsAggregator creates a new metrics aggregator.
func NewMetricsAggregator() *MetricsAggregator {
	return &MetricsAggregator{
		byEcosystem: make(map[string]*ecosystemAccumulator),
	}
}

// Add adds evaluation results to the aggregator.
func (ma *MetricsAggregator) Add(repoName, ecosystem string, expected, actual *architect.ProfileFragment) error {
	acc, ok := ma.byEcosystem[ecosystem]
	if !ok {
		acc = &ecosystemAccumulator{}
		ma.byEcosystem[ecosystem] = acc
	}

	// Compute import accuracy
	importFA := compareImportGraphs(expected.ImportGraph, actual.ImportGraph)
	acc.importTP += importFA.TruePositives
	acc.importFP += importFA.FalsePositives
	acc.importFN += importFA.FalseNegatives

	// Compute style hypothesis accuracy
	styleFA := compareStyles(expected, actual)
	acc.styleTP += styleFA.TruePositives
	acc.styleFP += styleFA.FalsePositives
	acc.styleFN += styleFA.FalseNegatives

	// Compute C4 accuracy
	c4FA := compareC4(expected, actual)
	acc.c4TP += c4FA.TruePositives
	acc.c4FP += c4FA.FalsePositives
	acc.c4FN += c4FA.FalseNegatives

	// Compute schema accuracy (if SQL analysis present)
	if expected.SQLAnalysis != nil || actual.SQLAnalysis != nil {
		schemaFA := compareSQL(expected.SQLAnalysis, actual.SQLAnalysis)
		acc.schemaTP += schemaFA.TruePositives
		acc.schemaFP += schemaFA.FalsePositives
		acc.schemaFN += schemaFA.FalseNegatives
	}

	acc.sampleCount++
	return nil
}

// Compute computes the final metrics per ecosystem.
func (ma *MetricsAggregator) Compute() []EcosystemMetrics {
	var metrics []EcosystemMetrics

	for ecosystem, acc := range ma.byEcosystem {
		em := EcosystemMetrics{
			Ecosystem:    ecosystem,
			SampleCount:  acc.sampleCount,
			ImportF1:     computeF1FromCounts(acc.importTP, acc.importFP, acc.importFN),
			StyleF1:      computeF1FromCounts(acc.styleTP, acc.styleFP, acc.styleFN),
			C4F1:         computeF1FromCounts(acc.c4TP, acc.c4FP, acc.c4FN),
			SchemaF1:     computeF1FromCounts(acc.schemaTP, acc.schemaFP, acc.schemaFN),
		}

		// Import precision/recall
		importDenom := acc.importTP + acc.importFP
		if importDenom > 0 {
			em.ImportPrecision = float64(acc.importTP) / float64(importDenom)
		}
		importRecallDenom := acc.importTP + acc.importFN
		if importRecallDenom > 0 {
			em.ImportRecall = float64(acc.importTP) / float64(importRecallDenom)
		}

		// Style precision/recall
		styleDenom := acc.styleTP + acc.styleFP
		if styleDenom > 0 {
			em.StylePrecision = float64(acc.styleTP) / float64(styleDenom)
		}
		styleRecallDenom := acc.styleTP + acc.styleFN
		if styleRecallDenom > 0 {
			em.StyleRecall = float64(acc.styleTP) / float64(styleRecallDenom)
		}

		// C4 precision/recall
		c4Denom := acc.c4TP + acc.c4FP
		if c4Denom > 0 {
			em.C4Precision = float64(acc.c4TP) / float64(c4Denom)
		}
		c4RecallDenom := acc.c4TP + acc.c4FN
		if c4RecallDenom > 0 {
			em.C4Recall = float64(acc.c4TP) / float64(c4RecallDenom)
		}

		// Schema precision/recall (only if we have SQL data)
		if acc.schemaTP > 0 || acc.schemaFP > 0 || acc.schemaFN > 0 {
			schemaDenom := acc.schemaTP + acc.schemaFP
			if schemaDenom > 0 {
				em.SchemaPrecision = float64(acc.schemaTP) / float64(schemaDenom)
			}
			schemaRecallDenom := acc.schemaTP + acc.schemaFN
			if schemaRecallDenom > 0 {
				em.SchemaRecall = float64(acc.schemaTP) / float64(schemaRecallDenom)
			}
		}

		// Overall score is average of all F1 scores
		f1Sum := em.ImportF1 + em.StyleF1 + em.C4F1
		f1Count := 3.0
		if em.SchemaF1 > 0 {
			f1Sum += em.SchemaF1
			f1Count++
		}
		em.OverallScore = f1Sum / f1Count

		metrics = append(metrics, em)
	}

	return metrics
}

// computeF1FromCounts computes F1 from TP/FP/FN counts.
func computeF1FromCounts(tp, fp, fn int) float64 {
	if tp == 0 && fp == 0 && fn == 0 {
		return 1.0 // Perfect: nothing expected, nothing produced
	}

	precision := float64(tp) / float64(tp+fp)
	recall := float64(tp) / float64(tp+fn)

	if precision+recall == 0 {
		return 0.0
	}
	return 2 * (precision * recall) / (precision + recall)
}

// compareStyles compares style hypotheses between expected and actual.
func compareStyles(expected, actual *architect.ProfileFragment) FieldAccuracy {
	// For now, styles are not directly in ProfileFragment
	// This is a placeholder for when we add style detection
	// In the current architecture, styles come from ArchitectureReport
	return FieldAccuracy{
		FieldName:     "style_hypothesis",
		TruePositives: 0,
		FalsePositives: 0,
		FalseNegatives: 0,
		MatchType:     "exact",
	}
}

// compareC4 compares C4 model elements between expected and actual.
func compareC4(expected, actual *architect.ProfileFragment) FieldAccuracy {
	fa := FieldAccuracy{
		FieldName: "c4_model",
		MatchType: "exact",
	}

	// Compare containers
	expectedContainers := make(map[string]bool)
	if expected.Infra != nil {
		for _, c := range expected.Infra.Containers {
			expectedContainers[c.Name] = true
		}
	}
	actualContainers := make(map[string]bool)
	if actual.Infra != nil {
		for _, c := range actual.Infra.Containers {
			actualContainers[c.Name] = true
		}
	}

	// True positives: containers in both
	for name := range expectedContainers {
		if actualContainers[name] {
			fa.TruePositives++
		} else {
			fa.FalseNegatives++
		}
	}

	// False positives: containers only in actual
	for name := range actualContainers {
		if !expectedContainers[name] {
			fa.FalsePositives++
		}
	}

	return fa
}

// CheckThresholds checks if metrics meet the exit criteria thresholds.
func CheckThresholds(metrics EcosystemMetrics) (bool, string) {
	var failures []string

	// Go ecosystem: >90% import, >85% style
	if metrics.Ecosystem == "go" {
		if metrics.ImportF1 < 0.90 {
			failures = append(failures, fmt.Sprintf("import F1 %.2f < 0.90", metrics.ImportF1))
		}
		if metrics.StyleF1 < 0.85 {
			failures = append(failures, fmt.Sprintf("style F1 %.2f < 0.85", metrics.StyleF1))
		}
	}

	// Python/Java: >65% import, >75% style
	if metrics.Ecosystem == "python" || metrics.Ecosystem == "java" {
		if metrics.ImportF1 < 0.65 {
			failures = append(failures, fmt.Sprintf("import F1 %.2f < 0.65", metrics.ImportF1))
		}
		if metrics.StyleF1 < 0.75 {
			failures = append(failures, fmt.Sprintf("style F1 %.2f < 0.75", metrics.StyleF1))
		}
	}

	// TypeScript/JavaScript: >65% import, >70% style
	if metrics.Ecosystem == "typescript" || metrics.Ecosystem == "javascript" {
		if metrics.ImportF1 < 0.65 {
			failures = append(failures, fmt.Sprintf("import F1 %.2f < 0.65", metrics.ImportF1))
		}
		if metrics.StyleF1 < 0.70 {
			failures = append(failures, fmt.Sprintf("style F1 %.2f < 0.70", metrics.StyleF1))
		}
	}

	// SQL: >80% schema
	if metrics.Ecosystem == "sql" && metrics.SchemaF1 > 0 {
		if metrics.SchemaF1 < 0.80 {
			failures = append(failures, fmt.Sprintf("schema F1 %.2f < 0.80", metrics.SchemaF1))
		}
	}

	if len(failures) > 0 {
		return false, fmt.Sprintf("threshold failures: %s", joinReasons(failures))
	}

	return true, ""
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	result := reasons[0]
	for i := 1; i < len(reasons); i++ {
		result += "; " + reasons[i]
	}
	return result
}

// FormatMetricsReport formats the metrics as a human-readable report.
func FormatMetricsReport(metrics []EcosystemMetrics) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%-15s  %8s  %8s  %8s  %8s  %8s  %8s  %8s  %8s\n",
		"Ecosystem", "ImpF1", "StyF1", "C4F1", "SchF1", "Overall", "Samples", "Status", "Details")
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 120))

	for _, m := range metrics {
		passed, reason := CheckThresholds(m)
		status := "PASS"
		if !passed {
			status = "FAIL"
		}

		fmt.Fprintf(&b, "%-15s  %8.3f  %8.3f  %8.3f  %8.3f  %8.3f  %8d  %8s",
			m.Ecosystem, m.ImportF1, m.StyleF1, m.C4F1, m.SchemaF1, m.OverallScore, m.SampleCount, status)

		if !passed {
			fmt.Fprintf(&b, "  %s", reason)
		}
		fmt.Fprintf(&b, "\n")
	}

	return b.String()
}

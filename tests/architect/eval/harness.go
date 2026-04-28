package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// GoldenRepo represents a known repository with expected results.
type GoldenRepo struct {
	Name             string   `json:"name"`
	URL              string   `json:"url,omitempty"`
	LocalPath        string   `json:"local_path"` // for local testing
	ExpectedStyles   []string `json:"expected_styles"`   // e.g. ["microservices", "event_driven"]
	ExpectedLangs    []string `json:"expected_languages"` // e.g. ["go", "python"]
	Containers       int      `json:"expected_containers"` // expected container count (approximate)
	HasContracts     bool     `json:"has_contracts"`
	Complexity       string   `json:"complexity"` // "simple", "medium", "complex"
}

// EvalMetrics holds scores for a single evaluation run.
type EvalMetrics struct {
	RepoName       string  `json:"repo_name"`
	StylePrecision float64 `json:"style_precision"`  // correct styles / predicted styles
	StyleRecall    float64 `json:"style_recall"`     // correct styles / expected styles
	StyleF1        float64 `json:"style_f1"`
	ImportAccuracy float64 `json:"import_accuracy"`  // estimated from confidence
	C4Completeness float64 `json:"c4_completeness"`  // containers found / expected containers
	LangAccuracy   float64 `json:"language_accuracy"` // correct languages / all expected
	OverallScore   float64 `json:"overall_score"`
	DurationS      float64 `json:"duration_s"`
	Error          string  `json:"error,omitempty"`
}

// EvalThresholds defines acceptable scores per metric.
type EvalThresholds struct {
	StylePrecision float64 `json:"style_precision_min"`
	StyleRecall    float64 `json:"style_recall_min"`
	C4Completeness float64 `json:"c4_completeness_min"`
	LangAccuracy   float64 `json:"language_accuracy_min"`
}

// DefaultThresholds returns the standard thresholds.
func DefaultThresholds() EvalThresholds {
	return EvalThresholds{
		StylePrecision: 0.6,
		StyleRecall:    0.5,
		C4Completeness: 0.5,
		LangAccuracy:   0.8,
	}
}

// EvalResult holds the result of evaluating against all golden repos.
type EvalResult struct {
	Timestamp  string        `json:"timestamp"`
	TotalRepos int           `json:"total_repos"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Metrics    []EvalMetrics `json:"metrics"`
	Thresholds EvalThresholds `json:"thresholds"`
}

// Harness runs evaluations against golden repos.
type Harness struct {
	repos      []GoldenRepo
	thresholds EvalThresholds
}

// NewHarness creates a new evaluation harness.
func NewHarness(repos []GoldenRepo, thresholds EvalThresholds) *Harness {
	return &Harness{
		repos:      repos,
		thresholds: thresholds,
	}
}

// RunLocal executes the evaluation against golden repos with local paths.
func (h *Harness) RunLocal(repoRoot string) (*EvalResult, error) {
	result := &EvalResult{
		Timestamp:  time.Now().Format(time.RFC3339),
		TotalRepos: len(h.repos),
		Thresholds: h.thresholds,
		Metrics:    make([]EvalMetrics, 0, len(h.repos)),
	}

	for _, repo := range h.repos {
		if repo.LocalPath == "" {
			// Skip repos without a local path
			continue
		}

		metrics := h.evaluateRepo(repo)
		result.Metrics = append(result.Metrics, metrics)

		if h.Passes(metrics) {
			result.Passed++
		} else {
			result.Failed++
		}
	}

	return result, nil
}

// evaluateRepo computes metrics for a single golden repo.
func (h *Harness) evaluateRepo(repo GoldenRepo) EvalMetrics {
	start := time.Now()
	metrics := EvalMetrics{
		RepoName:  repo.Name,
		DurationS: 0,
	}

	// Load the architecture report
	reportPath := filepath.Join(repo.LocalPath, ".sdp", "architecture", "report.json")
	report, err := h.loadReport(reportPath)
	if err != nil {
		metrics.Error = fmt.Sprintf("failed to load report: %v", err)
		metrics.DurationS = time.Since(start).Seconds()
		return metrics
	}

	// Compute style metrics
	precision, recall := h.computeStyleMetrics(report.StyleHypothesis.Styles, repo.ExpectedStyles)
	metrics.StylePrecision = precision
	metrics.StyleRecall = recall
	metrics.StyleF1 = computeF1(precision, recall)

	// Compute C4 completeness
	metrics.C4Completeness = h.computeC4Completeness(report, repo.Containers)

	// Compute language accuracy
	metrics.LangAccuracy = h.computeLangAccuracy(report.Languages, repo.ExpectedLangs)

	// Import accuracy is estimated from confidence
	metrics.ImportAccuracy = report.ConfidenceSummary.StructuralAnalysis

	// Overall score is the average of all metrics
	metrics.OverallScore = (metrics.StylePrecision + metrics.StyleRecall + metrics.StyleF1 +
		metrics.ImportAccuracy + metrics.C4Completeness + metrics.LangAccuracy) / 6.0

	metrics.DurationS = time.Since(start).Seconds()
	return metrics
}

// loadReport loads an ArchitectureReport from disk.
func (h *Harness) loadReport(path string) (*architect.ArchitectureReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var report architect.ArchitectureReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &report, nil
}

// computeStyleMetrics calculates precision and recall for style detection.
func (h *Harness) computeStyleMetrics(predicted []architect.StyleScore, expected []string) (precision, recall float64) {
	if len(predicted) == 0 && len(expected) == 0 {
		return 1.0, 1.0
	}
	if len(predicted) == 0 {
		return 0.0, 0.0
	}
	if len(expected) == 0 {
		return 0.0, 1.0
	}

	// Count correct predictions (style in expected list with confidence > 0.3)
	correct := 0
	for _, pred := range predicted {
		if pred.Confidence > 0.3 {
			for _, exp := range expected {
				if string(pred.Style) == exp {
					correct++
					break
				}
			}
		}
	}

	// Count high-confidence predictions
	totalPredicted := 0
	for _, pred := range predicted {
		if pred.Confidence > 0.3 {
			totalPredicted++
		}
	}

	precision = float64(correct) / float64(totalPredicted)
	recall = float64(correct) / float64(len(expected))

	return precision, recall
}

// computeC4Completeness calculates how well containers were detected.
func (h *Harness) computeC4Completeness(report *architect.ArchitectureReport, expectedContainers int) float64 {
	if expectedContainers == 0 {
		// If no containers expected, any detection is wrong
		if report.Metrics.ContainersDetected == 0 {
			return 1.0
		}
		return 0.0
	}

	completeness := float64(report.Metrics.ContainersDetected) / float64(expectedContainers)
	if completeness > 1.0 {
		return 1.0
	}
	return completeness
}

// computeLangAccuracy calculates language detection accuracy.
func (h *Harness) computeLangAccuracy(detected architect.LanguageInfo, expected []string) float64 {
	if len(expected) == 0 {
		return 1.0
	}

	correct := 0
	for _, expLang := range expected {
		for _, detLang := range detected.All {
			if expLang == detLang {
				correct++
				break
			}
		}
	}

	return float64(correct) / float64(len(expected))
}

// computeF1 calculates the F1 score from precision and recall.
func computeF1(precision, recall float64) float64 {
	if precision == 0 && recall == 0 {
		return 0.0
	}
	return 2 * (precision * recall) / (precision + recall)
}

// Passes checks if metrics meet all thresholds.
func (h *Harness) Passes(metrics EvalMetrics) bool {
	if metrics.Error != "" {
		return false
	}

	return metrics.StylePrecision >= h.thresholds.StylePrecision &&
		metrics.StyleRecall >= h.thresholds.StyleRecall &&
		metrics.C4Completeness >= h.thresholds.C4Completeness &&
		metrics.LangAccuracy >= h.thresholds.LangAccuracy
}

// WriteReport writes an EvalResult to a JSON file.
func WriteReport(result *EvalResult, path string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Package main implements a benchmark harness for the sdp microfirst micro-classifiers.
// It runs all four classifiers (wsverdict, bdseverity, bdtype, routing) against
// synthetic corpora and produces JSON evidence + a markdown report.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"sdp_dev/internal/inference/decompose"
	"sdp_dev/internal/inference/microfirst/bdtype"
	"sdp_dev/internal/inference/microfirst/knn"
	"sdp_dev/internal/inference/microfirst/routing"
	"sdp_dev/internal/inference/microfirst/wsverdict"
)

// mockEmbedder returns deterministic fake embeddings based on keyword hashing.
// Allows bench to run without a real Ollama instance.
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	h := fnv.New32a()
	h.Write([]byte(text))
	seed := int64(h.Sum32())
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec
	vec := make([]float64, 8)
	var norm float64
	for i := range vec {
		vec[i] = rng.NormFloat64()
		norm += vec[i] * vec[i]
	}
	norm = math.Sqrt(norm)
	for i := range vec {
		vec[i] /= norm
	}
	return vec, nil
}

// BenchResult holds the benchmark metrics for a single classifier.
type BenchResult struct {
	Classifier      string  `json:"classifier"`
	TotalRequests   int     `json:"total_requests"`
	MicroHandled    int     `json:"micro_handled"`
	Escalated       int     `json:"escalated"`
	FallbackRate    float64 `json:"fallback_rate"`
	P50Ms           float64 `json:"p50_ms"`
	P95Ms           float64 `json:"p95_ms"`
	Accuracy        float64 `json:"accuracy"`
	LLMCallsSaved   float64 `json:"llm_calls_saved_pct"`
	EstTokenSavings float64 `json:"est_token_savings_pct"`
}

// wsVerdictCase is a test case for the wsverdict classifier.
type wsVerdictCase struct {
	Input           wsverdict.WsVerdictInput `json:"input"`
	ExpectedVerdict string                   `json:"expected_verdict"`
}

// textCase is a test case for embedding-based classifiers.
type textCase struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Expected    string `json:"-"`
}

func main() {
	classifierFlag := flag.String("classifier", "all", "Classifier to bench: all|wsverdict|bdseverity|bdtype|routing")
	nFlag := flag.Int("n", 30, "Minimum samples per classifier")
	mockLatencyFlag := flag.Duration("mock-llm-latency", 800*time.Millisecond, "Simulated LLM latency")
	outputFlag := flag.String("output", "internal/build/.sdp/evidence/f147/", "Output directory")
	formatFlag := flag.String("format", "both", "Output format: json|markdown|both")
	flag.Parse()

	_ = mockLatencyFlag // used conceptually; bench measures micro-only latency

	ctx := context.Background()

	if err := os.MkdirAll(*outputFlag, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	var results []BenchResult

	run := func(name string) bool {
		return *classifierFlag == "all" || *classifierFlag == name
	}

	dataDir := filepath.Join("cmd", "sdp-microfirst-bench", "testdata")

	if run("wsverdict") {
		r, err := benchWsVerdict(ctx, dataDir, *nFlag)
		if err != nil {
			log.Fatalf("wsverdict bench: %v", err)
		}
		results = append(results, r)
	}

	if run("bdseverity") {
		r, err := benchBdSeverity(ctx, dataDir, *nFlag)
		if err != nil {
			log.Fatalf("bdseverity bench: %v", err)
		}
		results = append(results, r)
	}

	if run("bdtype") {
		r, err := benchBdType(ctx, dataDir, *nFlag)
		if err != nil {
			log.Fatalf("bdtype bench: %v", err)
		}
		results = append(results, r)
	}

	if run("routing") {
		r, err := benchRouting(ctx, dataDir, *nFlag)
		if err != nil {
			log.Fatalf("routing bench: %v", err)
		}
		results = append(results, r)
	}

	if *formatFlag == "json" || *formatFlag == "both" {
		for _, r := range results {
			path := filepath.Join(*outputFlag, r.Classifier+".json")
			if err := writeJSON(path, r); err != nil {
				log.Fatalf("write json %s: %v", path, err)
			}
			fmt.Printf("Wrote %s\n", path)
		}
	}

	if *formatFlag == "markdown" || *formatFlag == "both" {
		path := filepath.Join(*outputFlag, "report.md")
		if err := writeMarkdown(path, results); err != nil {
			log.Fatalf("write report.md: %v", err)
		}
		fmt.Printf("Wrote %s\n", path)
	}
}

// benchWsVerdict benchmarks the wsverdict classifier using rule-based evaluation (no embeddings).
func benchWsVerdict(_ context.Context, dataDir string, minN int) (BenchResult, error) {
	cases, err := loadWsVerdictCases(filepath.Join(dataDir, "wsverdict_cases.json"))
	if err != nil {
		return BenchResult{}, err
	}
	cases = extendToMin(cases, minN)

	clf := wsverdict.New(wsverdict.Default())
	var latencies []float64
	microHandled := 0
	correct := 0

	for _, tc := range cases {
		start := time.Now()
		result, _, err := clf.Run(context.Background(), tc.Input)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0 // ms
		if err != nil {
			return BenchResult{}, fmt.Errorf("wsverdict.Run: %w", err)
		}
		latencies = append(latencies, elapsed)

		if result.ConfStatus() == decompose.StatusOK {
			microHandled++
			if string(result.Verdict) == tc.ExpectedVerdict {
				correct++
			}
		}
	}

	return computeResult("wsverdict", len(cases), microHandled, latencies, correct), nil
}

// benchBdSeverity benchmarks the bdseverity classifier using a mock embedder.
func benchBdSeverity(ctx context.Context, dataDir string, minN int) (BenchResult, error) {
	cases, err := loadTextCasesJSONL(filepath.Join(dataDir, "bdseverity_cases.jsonl"), "expected_priority")
	if err != nil {
		return BenchResult{}, err
	}
	cases = extendTextCasesToMin(cases, minN)

	emb := &mockEmbedder{}
	const threshold = 0.85
	const topK = 5

	// Build kNN index from the first half as training corpus.
	split := len(cases) / 2
	if split < 5 {
		split = 5
	}
	trainCases := cases[:split]
	evalCases := cases[split:]
	if len(evalCases) == 0 {
		evalCases = cases
	}

	idx := knn.NewIndex[string]()
	for _, tc := range trainCases {
		text := tc.Title + ". " + tc.Description
		vec, err := emb.Embed(ctx, text)
		if err != nil {
			return BenchResult{}, err
		}
		idx.Add(vec, tc.Expected, tc.Title)
	}

	var latencies []float64
	microHandled := 0
	correct := 0

	for _, tc := range evalCases {
		text := tc.Title + ". " + tc.Description
		start := time.Now()
		vec, err := emb.Embed(ctx, text)
		if err != nil {
			return BenchResult{}, err
		}
		matches := idx.Query(vec, topK)
		result := knn.MajorityVote(matches, threshold)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		latencies = append(latencies, elapsed)

		if result.Status == decompose.StatusOK {
			microHandled++
			if result.Label == tc.Expected {
				correct++
			}
		}
	}

	return computeResult("bdseverity", len(evalCases), microHandled, latencies, correct), nil
}

// benchBdType benchmarks the bdtype classifier using a mock embedder.
func benchBdType(ctx context.Context, dataDir string, minN int) (BenchResult, error) {
	cases, err := loadTextCasesJSONL(filepath.Join(dataDir, "bdtype_cases.jsonl"), "expected_type")
	if err != nil {
		return BenchResult{}, err
	}
	cases = extendTextCasesToMin(cases, minN)

	emb := &mockEmbedder{}

	// Split into train/eval.
	split := len(cases) / 2
	if split < 5 {
		split = 5
	}
	trainCases := cases[:split]
	evalCases := cases[split:]
	if len(evalCases) == 0 {
		evalCases = cases
	}

	// Build corpus entries for bdtype.
	corpus := make([]bdtype.CorpusEntry, len(trainCases))
	for i, tc := range trainCases {
		corpus[i] = bdtype.CorpusEntry{
			ID:    fmt.Sprintf("corpus-%d", i),
			Text:  tc.Title + " " + tc.Description,
			Label: tc.Expected,
		}
	}

	clf, err := bdtype.NewBdTypeMicro(ctx, emb, corpus)
	if err != nil {
		return BenchResult{}, fmt.Errorf("build bdtype classifier: %w", err)
	}

	var latencies []float64
	microHandled := 0
	correct := 0

	for _, tc := range evalCases {
		in := bdtype.BdInput{Title: tc.Title, Description: tc.Description}
		start := time.Now()
		result, _, err := clf.Run(ctx, in)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		if err != nil {
			return BenchResult{}, fmt.Errorf("bdtype.Run: %w", err)
		}
		latencies = append(latencies, elapsed)

		if result.ConfStatus() == decompose.StatusOK {
			microHandled++
			if result.Type == tc.Expected {
				correct++
			}
		}
	}

	return computeResult("bdtype", len(evalCases), microHandled, latencies, correct), nil
}

// benchRouting benchmarks the routing classifier using a mock embedder.
func benchRouting(ctx context.Context, dataDir string, minN int) (BenchResult, error) {
	cases, err := loadTextCasesJSONL(filepath.Join(dataDir, "routing_cases.jsonl"), "expected_capability")
	if err != nil {
		return BenchResult{}, err
	}
	cases = extendTextCasesToMin(cases, minN)

	emb := &mockEmbedder{}

	// Split into train/eval.
	split := len(cases) / 2
	if split < 5 {
		split = 5
	}
	trainCases := cases[:split]
	evalCases := cases[split:]
	if len(evalCases) == 0 {
		evalCases = cases
	}

	// Build routing corpus from training cases + DefaultCorpus.
	defaultCorpus := routing.DefaultCorpus()
	corpusExamples := make([]routing.RoutingExample, 0, len(trainCases)+len(defaultCorpus))
	for _, tc := range trainCases {
		corpusExamples = append(corpusExamples, routing.RoutingExample{
			Title:       tc.Title,
			Description: tc.Description,
			Capability:  tc.Expected,
		})
	}
	corpusExamples = append(corpusExamples, defaultCorpus...)

	// Build kNN index directly (routing.New requires *embed.CachedEmbedder, so we build manually).
	const threshold = 0.80
	const topK = 3
	idx := knn.NewIndex[string]()
	for _, ex := range corpusExamples {
		text := ex.Title + " " + ex.Description
		vec, err := emb.Embed(ctx, text)
		if err != nil {
			return BenchResult{}, err
		}
		idx.Add(vec, ex.Capability, ex.Title)
	}

	var latencies []float64
	microHandled := 0
	correct := 0

	for _, tc := range evalCases {
		text := tc.Title + " " + tc.Description
		start := time.Now()
		vec, err := emb.Embed(ctx, text)
		if err != nil {
			return BenchResult{}, err
		}
		matches := idx.Query(vec, topK)
		result := knn.MajorityVote(matches, threshold)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		latencies = append(latencies, elapsed)

		if result.Status == decompose.StatusOK {
			microHandled++
			if result.Label == tc.Expected {
				correct++
			}
		}
	}

	return computeResult("routing", len(evalCases), microHandled, latencies, correct), nil
}

// computeResult calculates bench metrics from raw latency + outcome data.
func computeResult(name string, total, microHandled int, latencies []float64, correct int) BenchResult {
	escalated := total - microHandled

	var fallbackRate float64
	if total > 0 {
		fallbackRate = float64(escalated) / float64(total)
	}

	p50, p95 := percentiles(latencies)

	var accuracy float64
	if microHandled > 0 {
		accuracy = float64(correct) / float64(microHandled) * 100.0
	}

	var llmSaved, estTokenSavings float64
	if total > 0 {
		llmSaved = float64(microHandled) / float64(total) * 100.0
		estTokenSavings = llmSaved
	}

	return BenchResult{
		Classifier:      name,
		TotalRequests:   total,
		MicroHandled:    microHandled,
		Escalated:       escalated,
		FallbackRate:    fallbackRate,
		P50Ms:           p50,
		P95Ms:           p95,
		Accuracy:        accuracy,
		LLMCallsSaved:   llmSaved,
		EstTokenSavings: estTokenSavings,
	}
}

// percentiles returns p50 and p95 from a slice of latency values.
func percentiles(vals []float64) (p50, p95 float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)

	p50 = sorted[int(math.Floor(float64(len(sorted)-1)*0.50))]
	p95 = sorted[int(math.Floor(float64(len(sorted)-1)*0.95))]
	return p50, p95
}

// loadWsVerdictCases reads JSON array of wsVerdictCase from path.
func loadWsVerdictCases(path string) ([]wsVerdictCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load wsverdict cases: %w", err)
	}
	var cases []wsVerdictCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("parse wsverdict cases: %w", err)
	}
	return cases, nil
}

// rawTextCase is used for JSON/JSONL parsing of text-based cases.
type rawTextCase struct {
	Title               string `json:"title"`
	Description         string `json:"description"`
	ExpectedPriority    string `json:"expected_priority"`
	ExpectedType        string `json:"expected_type"`
	ExpectedCapability  string `json:"expected_capability"`
}

// loadTextCasesJSONL reads JSONL text cases and extracts the expected field by key.
func loadTextCasesJSONL(path, expectedKey string) ([]textCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load text cases %s: %w", path, err)
	}

	var cases []textCase
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var raw rawTextCase
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, fmt.Errorf("decode text case: %w", err)
		}
		var expected string
		switch expectedKey {
		case "expected_priority":
			expected = raw.ExpectedPriority
		case "expected_type":
			expected = raw.ExpectedType
		case "expected_capability":
			expected = raw.ExpectedCapability
		}
		cases = append(cases, textCase{
			Title:       raw.Title,
			Description: raw.Description,
			Expected:    expected,
		})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan text cases: %w", err)
	}
	return cases, nil
}

// extendToMin repeats wsverdict cases until we have at least minN.
func extendToMin(cases []wsVerdictCase, minN int) []wsVerdictCase {
	if len(cases) == 0 || len(cases) >= minN {
		return cases
	}
	result := make([]wsVerdictCase, 0, minN)
	for len(result) < minN {
		result = append(result, cases...)
	}
	return result[:minN]
}

// extendTextCasesToMin repeats text cases until we have at least minN.
func extendTextCasesToMin(cases []textCase, minN int) []textCase {
	if len(cases) == 0 || len(cases) >= minN {
		return cases
	}
	result := make([]textCase, 0, minN)
	for len(result) < minN {
		result = append(result, cases...)
	}
	return result[:minN]
}

// writeJSON serializes result to a JSON file.
func writeJSON(path string, r BenchResult) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

const reportTmpl = `# Microfirst Bench Report

Generated: {{ .Timestamp }}

## Classifier Metrics

| Classifier | Total | Micro Handled | Escalated | Fallback% | P50ms | P95ms | Accuracy% | LLM Saved% | Token Savings% |
|------------|-------|--------------|-----------|-----------|-------|-------|-----------|-----------|---------------|
{{ range .Results -}}
| {{ .Classifier }} | {{ .TotalRequests }} | {{ .MicroHandled }} | {{ .Escalated }} | {{ printf "%.1f" .FallbackRate100 }} | {{ printf "%.3f" .P50Ms }} | {{ printf "%.3f" .P95Ms }} | {{ printf "%.1f" .Accuracy }} | {{ printf "%.1f" .LLMCallsSaved }} | {{ printf "%.1f" .EstTokenSavings }} |
{{ end }}
## Summary

Total LLM calls saved: {{ printf "%.1f" .TotalLLMSaved }}%

{{ range .Results -}}
- **{{ .Classifier }}**: {{ .MicroHandled }}/{{ .TotalRequests }} handled by micro ({{ printf "%.1f" .LLMCallsSaved }}% LLM savings)
{{ end }}
`

type reportData struct {
	Timestamp    string
	Results      []reportRow
	TotalLLMSaved float64
}

type reportRow struct {
	BenchResult
	FallbackRate100 float64
}

// writeMarkdown renders the markdown report.
func writeMarkdown(path string, results []BenchResult) error {
	var totalSaved, count float64
	rows := make([]reportRow, len(results))
	for i, r := range results {
		rows[i] = reportRow{
			BenchResult:     r,
			FallbackRate100: r.FallbackRate * 100.0,
		}
		totalSaved += r.LLMCallsSaved
		count++
	}
	var avg float64
	if count > 0 {
		avg = totalSaved / count
	}

	tmpl, err := template.New("report").Parse(reportTmpl)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, reportData{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Results:      rows,
		TotalLLMSaved: avg,
	})
}

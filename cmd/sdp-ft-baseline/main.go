// Command sdp-ft-baseline runs N samples from eval.jsonl through a vanilla
// Ollama model (no fine-tune) and records prediction quality. The output is
// the reference point against which fine-tuned models are compared.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
	"github.com/fall-out-bug/sdp_lab/internal/finetune"
)

type baselineRecord struct {
	Index           int             `json:"index"`
	UserPrompt      string          `json:"user_prompt"`
	ExpectedLabel   finetune.Label  `json:"expected_label"`
	RawResponse     string          `json:"raw_response"`
	PredictedLabel  *finetune.Label `json:"predicted_label,omitempty"`
	ParseOK         bool            `json:"parse_ok"`
	ComplexityMatch bool            `json:"complexity_match"`
	TaskTypeMatch   bool            `json:"task_type_match"`
	RiskMatch       bool            `json:"risk_match"`
	AllMatch        bool            `json:"all_match"`
	LatencyMS       int64           `json:"latency_ms"`
}

type baselineReport struct {
	Model              string            `json:"model"`
	BaseURL            string            `json:"base_url"`
	Timestamp          string            `json:"timestamp"`
	N                  int               `json:"n"`
	ParseOKCount       int               `json:"parse_ok_count"`
	ComplexityAccuracy float64           `json:"complexity_accuracy"`
	TaskTypeAccuracy   float64           `json:"task_type_accuracy"`
	RiskAccuracy       float64           `json:"risk_accuracy"`
	AllMatchAccuracy   float64           `json:"all_match_accuracy"`
	Records            []baselineRecord  `json:"records"`
	Confusion          map[string]int    `json:"confusion_complexity"`
}

func main() {
	var (
		evalPath  = flag.String("eval", "internal/dispatch/training/eval.jsonl", "eval JSONL")
		outPath   = flag.String("out", "internal/dispatch/training/baseline.json", "where to save baseline report")
		model     = flag.String("model", "qwen2.5:3b", "Ollama model tag")
		baseURL   = flag.String("base-url", dispatch.DefaultOllamaBaseURL, "Ollama base URL")
		n         = flag.Int("n", 10, "how many eval samples to run")
		timeoutS  = flag.Int("timeout", 120, "per-request timeout (seconds)")
	)
	flag.Parse()

	samples, err := readJSONL(*evalPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read eval: %v\n", err)
		os.Exit(1)
	}
	if len(samples) < *n {
		fmt.Fprintf(os.Stderr, "eval has %d samples, requested %d\n", len(samples), *n)
		os.Exit(1)
	}
	samples = samples[:*n]

	client := dispatch.NewOllamaClient(*baseURL)
	client.HTTPClient.Timeout = time.Duration(*timeoutS) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutS**n)*time.Second)
	defer cancel()
	if err := client.HealthCheck(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ollama health: %v\n", err)
		os.Exit(1)
	}

	report := baselineReport{
		Model:     *model,
		BaseURL:   *baseURL,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		N:         *n,
		Confusion: map[string]int{},
	}

	for i, s := range samples {
		expected := parseAssistantLabel(s)
		userMsg := s.Messages[1].Content
		chat := []dispatch.ChatMessage{
			{Role: "system", Content: s.Messages[0].Content},
			{Role: "user", Content: userMsg},
		}

		t0 := time.Now()
		resp, err := client.Chat(ctx, *model, chat)
		lat := time.Since(t0).Milliseconds()

		rec := baselineRecord{
			Index:         i,
			UserPrompt:    userMsg,
			ExpectedLabel: expected,
			RawResponse:   resp,
			LatencyMS:     lat,
		}
		if err != nil {
			rec.RawResponse = "ERROR: " + err.Error()
			fmt.Fprintf(os.Stderr, "[%d] generate: %v\n", i, err)
			report.Records = append(report.Records, rec)
			continue
		}

		predicted, ok := tryParseLabel(resp)
		rec.ParseOK = ok
		if ok {
			rec.PredictedLabel = &predicted
			rec.ComplexityMatch = predicted.Complexity == expected.Complexity
			rec.TaskTypeMatch = predicted.TaskType == expected.TaskType
			rec.RiskMatch = predicted.Risk == expected.Risk
			rec.AllMatch = rec.ComplexityMatch && rec.TaskTypeMatch && rec.RiskMatch
			report.ParseOKCount++
			if rec.ComplexityMatch {
				report.ComplexityAccuracy++
			}
			if rec.TaskTypeMatch {
				report.TaskTypeAccuracy++
			}
			if rec.RiskMatch {
				report.RiskAccuracy++
			}
			if rec.AllMatch {
				report.AllMatchAccuracy++
			}
			key := fmt.Sprintf("%s->%s", expected.Complexity, predicted.Complexity)
			report.Confusion[key]++
		}
		report.Records = append(report.Records, rec)
		fmt.Printf("[%d] %dms parse=%v cmp=%v type=%v risk=%v\n",
			i, lat, ok, rec.ComplexityMatch, rec.TaskTypeMatch, rec.RiskMatch)
	}

	if *n > 0 {
		report.ComplexityAccuracy /= float64(*n)
		report.TaskTypeAccuracy /= float64(*n)
		report.RiskAccuracy /= float64(*n)
		report.AllMatchAccuracy /= float64(*n)
	}

	out, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create out: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nbaseline saved: %s\n", *outPath)
	fmt.Printf("parse_ok:           %d/%d\n", report.ParseOKCount, *n)
	fmt.Printf("complexity_acc:     %.2f\n", report.ComplexityAccuracy)
	fmt.Printf("task_type_acc:      %.2f\n", report.TaskTypeAccuracy)
	fmt.Printf("risk_acc:           %.2f\n", report.RiskAccuracy)
	fmt.Printf("all_match_acc:      %.2f\n", report.AllMatchAccuracy)
}

func readJSONL(path string) ([]finetune.Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []finetune.Sample
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var s finetune.Sample
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, sc.Err()
}

func parseAssistantLabel(s finetune.Sample) finetune.Label {
	var l finetune.Label
	_ = json.Unmarshal([]byte(s.Messages[2].Content), &l)
	return l
}

// tryParseLabel handles unfine-tuned models that wrap JSON in code fences,
// add prose, or emit malformed values. Strategy: locate the first {...}
// substring and json.Unmarshal that.
func tryParseLabel(raw string) (finetune.Label, bool) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return finetune.Label{}, false
	}
	var l finetune.Label
	if err := json.Unmarshal([]byte(raw[start:end+1]), &l); err != nil {
		return finetune.Label{}, false
	}
	return l, true
}

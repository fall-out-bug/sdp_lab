package finetune

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
)

// BuildOptions controls dataset assembly.
type BuildOptions struct {
	WSDir      string // docs/workstreams/backlog
	BeadsPath  string // .beads/issues.jsonl
	EvalRatio  float64
	Seed       int64
	MinSamples int
}

// BuildReport summarises what made it in and what was dropped.
type BuildReport struct {
	WSLoaded         int
	BeadsLoaded      int
	WSSkipped        map[string]int
	BeadsSkipped     map[string]int
	Duplicates       int
	TotalAfterDedup  int
	LabelDistribution map[string]int
}

// Build collects, deduplicates, and splits samples into train/eval.
func Build(opts BuildOptions) (Split, BuildReport, error) {
	if opts.EvalRatio <= 0 || opts.EvalRatio >= 1 {
		opts.EvalRatio = 0.2
	}
	if opts.MinSamples == 0 {
		opts.MinSamples = 50
	}

	report := BuildReport{
		LabelDistribution: map[string]int{},
	}

	wsSamples, wsSkip, err := LoadWorkstreams(opts.WSDir)
	if err != nil {
		return Split{}, report, fmt.Errorf("finetune: load ws: %w", err)
	}
	report.WSLoaded = len(wsSamples)
	report.WSSkipped = wsSkip

	beadsSamples, bSkip, err := LoadBeads(opts.BeadsPath)
	if err != nil {
		return Split{}, report, fmt.Errorf("finetune: load beads: %w", err)
	}
	report.BeadsLoaded = len(beadsSamples)
	report.BeadsSkipped = bSkip

	all := append(wsSamples, beadsSamples...)
	deduped := dedup(all)
	report.Duplicates = len(all) - len(deduped)
	report.TotalAfterDedup = len(deduped)

	for _, s := range deduped {
		k := fmt.Sprintf("%s/%s/%s", s.Meta.Label.Complexity, s.Meta.Label.TaskType, s.Meta.Label.Risk)
		report.LabelDistribution[k]++
	}

	if len(deduped) < opts.MinSamples {
		return Split{}, report, fmt.Errorf("finetune: only %d samples after dedup, need >=%d", len(deduped), opts.MinSamples)
	}

	split := splitTrainEval(deduped, opts.EvalRatio, opts.Seed)
	return split, report, nil
}

// dedup keeps the first sample per InputKey. Sort by source order to make
// the result deterministic.
func dedup(samples []Sample) []Sample {
	sort.SliceStable(samples, func(i, j int) bool {
		if samples[i].Meta.Source != samples[j].Meta.Source {
			return samples[i].Meta.Source < samples[j].Meta.Source
		}
		return samples[i].Meta.SourceID < samples[j].Meta.SourceID
	})
	seen := map[string]bool{}
	out := make([]Sample, 0, len(samples))
	for _, s := range samples {
		if seen[s.Meta.InputKey] {
			continue
		}
		seen[s.Meta.InputKey] = true
		out = append(out, s)
	}
	return out
}

// splitTrainEval shuffles deterministically (Seed) and slices into eval/train.
//
// Sizing rules (in order):
//  1. evalCount = round(len(samples) * evalRatio)
//  2. cap evalCount at len(samples)-1 so train always has at least 1 sample
//  3. if cap allows, lift evalCount to at least 10 (challenge minimum)
func splitTrainEval(samples []Sample, evalRatio float64, seed int64) Split {
	r := rand.New(rand.NewSource(seed))
	idx := r.Perm(len(samples))

	evalCount := int(float64(len(samples)) * evalRatio)
	maxEval := len(samples) - 1
	if maxEval < 0 {
		maxEval = 0
	}
	if evalCount < 10 && maxEval >= 10 {
		evalCount = 10
	}
	if evalCount > maxEval {
		evalCount = maxEval
	}

	eval := make([]Sample, 0, evalCount)
	train := make([]Sample, 0, len(samples)-evalCount)
	for i, k := range idx {
		if i < evalCount {
			eval = append(eval, samples[k])
		} else {
			train = append(train, samples[k])
		}
	}
	return Split{Train: train, Eval: eval}
}

func mustMarshalLabel(l Label) string {
	b, err := json.Marshal(l)
	if err != nil {
		// Label fields are plain strings — Marshal can't fail in practice.
		panic(err)
	}
	return string(b)
}

func hashStr(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

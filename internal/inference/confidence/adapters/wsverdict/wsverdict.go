// Package wsverdict wires confidence checking around ws-verdict JSON outputs.
// A ws-verdict gates the merge of a workstream — the wrong verdict ships a
// broken change or kills working code, so this is the highest-stakes
// call-site in the F144 set. Profile: full strategy stack (constraint,
// optional N-sample, full self-check critic). UNSURE → human handoff per
// Policy.UnsureBehavior.
package wsverdict

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"sdp_dev/internal/inference/confidence"
	"sdp_dev/internal/inference/confidence/constraint"
	"sdp_dev/internal/inference/confidence/nsample"
	"sdp_dev/internal/inference/confidence/selfcheck"
)

// Verdict mirrors the JSON schema in schema/ws-verdict.schema.json. It is the
// parsed shape that consumers of ws-verdicts type-check against.
type Verdict struct {
	WSID                string         `json:"ws_id"`
	FeatureID           string         `json:"feature_id"`
	Verdict             string         `json:"verdict"`
	Commit              string         `json:"commit,omitempty"`
	Timestamp           string         `json:"timestamp,omitempty"`
	QualityGates        QualityGates   `json:"quality_gates"`
	ACEvidence          []ACEvidence   `json:"ac_evidence,omitempty"`
	ExistingWorkSummary string         `json:"existing_work_summary"`
	ChangedFiles        []string       `json:"changed_files,omitempty"`
	TDDCycles           int            `json:"tdd_cycles,omitempty"`
	Extra               map[string]any `json:"-"`
}

type QualityGates struct {
	TestsPass         bool     `json:"tests_pass"`
	LintClean         bool     `json:"lint_clean"`
	CoveragePct       *float64 `json:"coverage_pct,omitempty"`
	CoverageThreshold *float64 `json:"coverage_threshold,omitempty"`
	MaxFileLOC        *int     `json:"max_file_loc,omitempty"`
	BuildOK           *bool    `json:"build_ok,omitempty"`
	VetOK             *bool    `json:"vet_ok,omitempty"`
}

type ACEvidence struct {
	AC       string `json:"ac"`
	Met      bool   `json:"met"`
	Evidence string `json:"evidence,omitempty"`
}

// Options configures a ws-verdict Checker. SchemaJSON is the JSON-Schema
// source (typically schema/ws-verdict.schema.json contents). EnableNSample
// turns on the N-sample strategy for production-blocking decisions; default
// false because re-sampling a verdict requires the original prompt and an
// LLMCaller wired to the same model.
type Options struct {
	SchemaJSON    []byte
	Caller        confidence.LLMCaller
	EnableNSample bool
	NSamplePrompt string
	Policy        *confidence.Policy
}

// New constructs a confidence.Checker[Verdict] for ws-verdict gating. If
// Options.Policy is nil, the canonical full-set policy is used:
// thresholds 0.8/0.5, weights 0.4/0.4/0.2, UnsureBehavior=HumanHandoff.
func New(opts Options) (*confidence.Checker[Verdict], error) {
	if len(opts.SchemaJSON) == 0 {
		return nil, fmt.Errorf("wsverdict: SchemaJSON is required")
	}

	schemaValidator, err := compileSchema(opts.SchemaJSON)
	if err != nil {
		return nil, fmt.Errorf("wsverdict: %w", err)
	}

	// Wrap schema validator with semantic-consistency hard checks. These
	// catch contradictions that JSON-Schema cannot express (e.g. PASS
	// claimed alongside tests_pass=false). They are hard-fail because a
	// self-contradicting verdict is not a "low-confidence answer" — it's
	// invalid output.
	hardSchema := func(raw string) error {
		if err := schemaValidator(raw); err != nil {
			return err
		}
		return checkSemanticConsistency(raw)
	}

	cs, err := constraint.New[Verdict](constraint.Options[Verdict]{
		SchemaValidator: hardSchema,
		Invariants:      defaultInvariants(),
	})
	if err != nil {
		return nil, fmt.Errorf("wsverdict: build constraint: %w", err)
	}

	strategies := []confidence.Strategy[Verdict]{cs}

	if opts.Caller != nil {
		sc, err := selfcheck.New[Verdict](selfcheck.Options[Verdict]{
			Mode: selfcheck.ModeFull,
		})
		if err != nil {
			return nil, fmt.Errorf("wsverdict: build self-check: %w", err)
		}
		strategies = append(strategies, sc)
	}

	if opts.EnableNSample {
		if opts.Caller == nil {
			return nil, fmt.Errorf("wsverdict: EnableNSample requires Caller")
		}
		ns, err := nsample.New[Verdict](nsample.Options[Verdict]{
			Temperatures: []float64{0.0, 0.3, 0.7},
			BasePrompt:   opts.NSamplePrompt,
			Parser:       parseVerdict,
			Agreement:    verdictAgreement,
		})
		if err != nil {
			return nil, fmt.Errorf("wsverdict: build nsample: %w", err)
		}
		strategies = append(strategies, ns)
	}

	policy := canonicalPolicy()
	if opts.Policy != nil {
		policy = *opts.Policy
	}

	return confidence.NewChecker[Verdict](opts.Caller, strategies, policy)
}

// Verify is a one-shot helper that parses raw, then runs the Checker.
// Returns the confidence.Result[Verdict] for inspection (Status, Score,
// SubScores, Reasons, Trace). input must be the original prompt/evidence that
// produced rawJSON; self-check and N-sample strategies need it to validate the
// answer against the actual task rather than against itself.
func Verify(ctx context.Context, checker *confidence.Checker[Verdict], input string, rawJSON []byte) (confidence.Result[Verdict], error) {
	v, perr := parseVerdict(string(rawJSON))
	// We pass the parsed Verdict (or zero value) regardless — the
	// constraint strategy will hard-fail on schema violation, surfacing
	// the parse error there. Returning early on parse error would deny
	// callers visibility into Trace.
	answer := v
	rawStr := string(rawJSON)
	if perr != nil {
		// Parse failure becomes a hard schema fail in constraint output.
		// Don't return here — the Checker will catch it.
		_ = perr
	}
	return checker.Check(ctx, confidence.Request[Verdict]{
		Input:  input,
		Answer: answer,
		Raw:    rawStr,
	})
}

// canonicalPolicy returns the F144 ws-verdict default — full set with human
// handoff routing for UNSURE.
func canonicalPolicy() confidence.Policy {
	p := confidence.DefaultPolicy()
	p.UnsureBehavior = confidence.UnsureHumanHandoff
	return p
}

func compileSchema(schemaJSON []byte) (func(string) error, error) {
	c := jsonschema.NewCompiler()
	if err := c.AddResource("ws-verdict.schema.json", bytes.NewReader(schemaJSON)); err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	sch, err := c.Compile("ws-verdict.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	return func(raw string) error {
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		return sch.Validate(v)
	}, nil
}

// parseVerdict deserializes a ws-verdict JSON document into Verdict.
func parseVerdict(raw string) (Verdict, error) {
	var v Verdict
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return Verdict{}, fmt.Errorf("ws-verdict: parse: %w", err)
	}
	return v, nil
}

// verdictAgreement returns mean pairwise equality on the .Verdict field.
// We don't compare full structs because timestamps and reasoning differ
// per sample — we want consensus on the gating decision itself.
func verdictAgreement(samples []Verdict) float64 {
	n := len(samples)
	if n < 2 {
		if n == 1 {
			return 1
		}
		return 0
	}
	pairs, matches := 0, 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairs++
			if samples[i].Verdict == samples[j].Verdict {
				matches++
			}
		}
	}
	if pairs == 0 {
		return 0
	}
	return float64(matches) / float64(pairs)
}

// checkSemanticConsistency hard-fails verdicts that contradict themselves.
// JSON-Schema cannot encode "PASS implies tests_pass=true" — these checks
// fill that gap. Returning a non-nil error here triggers constraint hard-fail
// (Status=FAIL forced), which is correct for self-contradicting output.
func checkSemanticConsistency(raw string) error {
	v, err := parseVerdict(raw)
	if err != nil {
		// Already reported by schema validator's parse step; defensive.
		return nil
	}
	if v.Verdict == "PASS" {
		if !v.QualityGates.TestsPass {
			return fmt.Errorf("verdict=PASS contradicts tests_pass=false")
		}
		if !v.QualityGates.LintClean {
			return fmt.Errorf("verdict=PASS contradicts lint_clean=false")
		}
		for _, ac := range v.ACEvidence {
			if !ac.Met {
				return fmt.Errorf("verdict=PASS but AC %q is not met", ac.AC)
			}
		}
	}
	return nil
}

var wsIDPattern = regexp.MustCompile(`^\d{2}-\d{3}-\d{2}$`)

// defaultInvariants captures structural rules that go beyond JSON-Schema —
// cross-field consistency the schema cannot express.
func defaultInvariants() []constraint.Invariant[Verdict] {
	return []constraint.Invariant[Verdict]{
		{
			Name: "ws-id-format",
			Check: func(v Verdict) (bool, string) {
				if v.WSID == "" {
					return false, "empty ws_id"
				}
				if !wsIDPattern.MatchString(v.WSID) {
					return false, fmt.Sprintf("ws_id %q does not match NN-NNN-NN", v.WSID)
				}
				return true, ""
			},
		},
		{
			Name: "verdict-known",
			Check: func(v Verdict) (bool, string) {
				switch v.Verdict {
				case "PASS", "FAIL", "PARTIAL":
					return true, ""
				default:
					return false, "unknown verdict " + v.Verdict
				}
			},
		},
		{
			Name: "pass-requires-tests",
			Check: func(v Verdict) (bool, string) {
				if v.Verdict == "PASS" && !v.QualityGates.TestsPass {
					return false, "PASS but tests_pass=false"
				}
				return true, ""
			},
		},
		{
			Name: "pass-requires-lint",
			Check: func(v Verdict) (bool, string) {
				if v.Verdict == "PASS" && !v.QualityGates.LintClean {
					return false, "PASS but lint_clean=false"
				}
				return true, ""
			},
		},
		{
			Name: "ac-not-met-implies-not-pass",
			Check: func(v Verdict) (bool, string) {
				if v.Verdict != "PASS" {
					return true, ""
				}
				for _, ac := range v.ACEvidence {
					if !ac.Met {
						return false, "PASS with unmet AC: " + ac.AC
					}
				}
				return true, ""
			},
		},
	}
}

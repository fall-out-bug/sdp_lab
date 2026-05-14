package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type verdictDoc struct {
	Verdict         string       `json:"verdict"`
	P0Count         *int         `json:"p0_count,omitempty"`
	P1Count         *int         `json:"p1_count,omitempty"`
	FindingIDs      []string     `json:"finding_ids,omitempty"`
	BlockingIDs     []string     `json:"blocking_ids,omitempty"`
	OverrideReason  string       `json:"override_reason,omitempty"`
	EscalationIssue string       `json:"escalation_issue,omitempty"`
	FindingsDetail  []findingDoc `json:"findings_detail,omitempty"`
	ModelPanel      []modelDoc   `json:"model_panel,omitempty"`
	ReviewerRuntime string       `json:"reviewer_runtime,omitempty"`
}

type findingDoc struct {
	Priority string `json:"priority,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type modelDoc struct {
	Slot            string `json:"slot"`
	Status          string `json:"status"`
	AssessmentState string `json:"assessment_state,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	fs := flag.NewFlagSet("sdp-review-verdict-validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	schemaPath := fs.String("schema", "schema/review-verdict.schema.json", "review verdict JSON schema")
	requireApproval := fs.Bool("require-approval", false, "fail unless verdict is approval-capable or explicitly overridden")
	format := fs.String("format", "text", "output format: text or policy-json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: sdp-review-verdict-validate [--schema schema/review-verdict.schema.json] [--require-approval] [--format text|policy-json] <review_verdict.json>")
		return 2
	}
	if *format != "text" && *format != "policy-json" {
		fmt.Fprintf(stderr, "unsupported format %q\n", *format)
		return 2
	}

	path := fs.Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "read verdict: %v\n", err)
		return 1
	}
	if err := validateSchema(*schemaPath, data); err != nil {
		fmt.Fprintf(stderr, "schema validation failed: %v\n", err)
		return 1
	}
	var verdict verdictDoc
	if err := json.Unmarshal(data, &verdict); err != nil {
		fmt.Fprintf(stderr, "parse verdict: %v\n", err)
		return 1
	}

	policy := buildPolicy(verdict)
	if *format == "policy-json" {
		out, err := json.Marshal(policy)
		if err != nil {
			fmt.Fprintf(stderr, "marshal policy: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(out))
		return 0
	}

	if *requireApproval {
		if err := requireApprovalCapable(verdict, policy); err != nil {
			fmt.Fprintf(stderr, "not approval-capable: %v\n", err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "valid")
	return 0
}

func validateSchema(schemaPath string, data []byte) error {
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema %s: %w", schemaPath, err)
	}
	compiler := jsonschema.NewCompiler()
	name := filepath.Base(schemaPath)
	if err := compiler.AddResource(name, bytes.NewReader(schemaData)); err != nil {
		return err
	}
	schema, err := compiler.Compile(name)
	if err != nil {
		return err
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	return schema.Validate(doc)
}

type policySummary struct {
	P0Findings               int  `json:"p0_findings"`
	P1Findings               int  `json:"p1_findings"`
	P2Findings               int  `json:"p2_findings"`
	ReviewEscalated          bool `json:"review_escalated"`
	ReviewCannotVerify       bool `json:"review_cannot_verify"`
	ApprovalCapable          bool `json:"approval_capable"`
	MaintainerOverride       bool `json:"maintainer_override"`
	ReviewFindingsReferenced bool `json:"review_findings_referenced"`
}

func buildPolicy(v verdictDoc) policySummary {
	p := policySummary{
		ReviewEscalated:          strings.EqualFold(v.Verdict, "ESCALATED"),
		MaintainerOverride:       strings.TrimSpace(v.OverrideReason) != "",
		ReviewFindingsReferenced: len(v.FindingIDs) > 0 || len(v.BlockingIDs) > 0,
	}
	if v.P0Count != nil {
		p.P0Findings = *v.P0Count
	}
	if v.P1Count != nil {
		p.P1Findings = *v.P1Count
	}
	for _, f := range v.FindingsDetail {
		priority := strings.ToUpper(firstNonEmpty(f.Priority, f.Severity))
		switch priority {
		case "P0":
			if v.P0Count == nil {
				p.P0Findings++
			}
		case "P1":
			if v.P1Count == nil {
				p.P1Findings++
			}
		case "P2":
			p.P2Findings++
		}
	}
	if v.P1Count == nil && len(v.BlockingIDs) > p.P1Findings {
		p.P1Findings = len(v.BlockingIDs)
	}
	requiredModels := 0
	okModels := 0
	cannotVerify := false
	requiredSlots := map[string]bool{"zai": false, "kimi": false, "minimax": false}
	for _, m := range v.ModelPanel {
		requiredModels++
		if _, ok := requiredSlots[m.Slot]; ok {
			requiredSlots[m.Slot] = true
		}
		state := strings.ToLower(strings.TrimSpace(m.AssessmentState))
		if m.Status == "ok" && state == "assessed" {
			okModels++
		}
		if m.Status != "ok" || state != "assessed" {
			cannotVerify = true
		}
	}
	for _, present := range requiredSlots {
		if !present {
			cannotVerify = true
		}
	}
	p.ReviewCannotVerify = cannotVerify || p.ReviewEscalated
	p.ApprovalCapable = strings.EqualFold(v.Verdict, "APPROVED") && !p.ReviewCannotVerify && p.P0Findings == 0 && p.P1Findings == 0
	if requiredModels != len(requiredSlots) || okModels != len(requiredSlots) {
		p.ApprovalCapable = false
	}
	return p
}

func requireApprovalCapable(v verdictDoc, p policySummary) error {
	if p.ApprovalCapable {
		return nil
	}
	if p.MaintainerOverride && p.ReviewEscalated && strings.TrimSpace(v.EscalationIssue) != "" {
		return nil
	}
	if p.ReviewEscalated {
		return fmt.Errorf("verdict is ESCALATED; maintainer override is required")
	}
	if p.P0Findings > 0 || p.P1Findings > 0 {
		return fmt.Errorf("blocking findings remain: P0=%d P1=%d", p.P0Findings, p.P1Findings)
	}
	if p.ReviewCannotVerify {
		return fmt.Errorf("review cannot verify required model panel")
	}
	return fmt.Errorf("verdict %q is not approval-capable", v.Verdict)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveWorkstream(labels []string) string {
	if hasLabel(labels, "workstream:policy-slugify-trim") {
		return "policy-slugify-trim"
	}
	if hasLabel(labels, "workstream:model-chain-default-fallback") {
		return "model-chain-default-fallback"
	}
	if hasLabel(labels, "workstream:policy-k8s-risk-high") {
		return "policy-k8s-risk-high"
	}
	if hasLabel(labels, "workstream:telegram-ingress-intake") {
		return "telegram-ingress-intake"
	}
	if hasLabel(labels, "workstream:planner-boundary-decomposition") {
		return "planner-boundary-decomposition"
	}
	if hasLabel(labels, "workstream:oneshot-swarm-orchestrator") {
		return "oneshot-swarm-orchestrator"
	}
	if hasLabel(labels, "workstream:handoff-validation") {
		return "handoff-validation"
	}
	if hasLabel(labels, "workstream:generic") {
		return "generic"
	}
	if hasLabel(labels, "workstream:self-improvement") {
		return "self-improvement"
	}
	if hasLabel(labels, "workstream:evaluator-recommendation") {
		return "evaluator-recommendation"
	}
	return ""
}

func applyWorkstreamFlow(workstream string, issueID string, issue issueDetail) []string {
	changedFiles := []string{}
	switch workstream {
	case "policy-slugify-trim":
		if err := patchSlugifyForTrim("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := addSlugifyRegressionTest("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/policy/decision.go", "internal/policy/decision_test.go"}
	case "model-chain-default-fallback":
		if err := patchModelChainUnknownFallback("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := addModelChainRegressionTest("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/policy/model_chain.go", "internal/policy/model_chain_test.go"}
	case "policy-k8s-risk-high":
		if err := patchRiskK8sHigh("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := addRiskK8sRegressionTest("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/policy/decision.go", "internal/policy/decision_test.go"}
	case "telegram-ingress-intake":
		if err := ensureTelegramIntakeFiles("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/intake/telegram.go", "internal/intake/telegram_test.go"}
	case "planner-boundary-decomposition":
		if err := ensurePlannerEnvelopeFiles("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/planner/envelope.go", "internal/planner/envelope_test.go"}
	case "oneshot-swarm-orchestrator":
		if err := ensureOneShotManifestFiles("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/oneshot/manifest.go", "internal/oneshot/manifest_test.go"}
	case "handoff-validation":
		if err := appendHandoffValidationTimestamp("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"docs/AGENT_HANDOFF.md"}
	case "generic":
		changedFiles = applyGenericWorkstream(".", issueID, issue)
	case "self-improvement":
		changedFiles = applySelfImprovementWorkstream(".", issueID, issue)
	case "evaluator-recommendation":
		changedFiles = applyEvaluatorRecommendationWorkstream(".", issueID, issue)
	}
	return changedFiles
}

func commitBodyForWorkstream(workstream string) string {
	switch workstream {
	case "policy-slugify-trim":
		return "Fix slugify truncation and add regression coverage."
	case "handoff-validation":
		return "Add handoff validation timestamp for adapter checklist run."
	case "generic":
		return "Generic workstream placeholder; full LLM delegation pending opencode-implement."
	case "self-improvement":
		return "Self-improvement cycle: log improvement task."
	case "evaluator-recommendation":
		return "Evaluator recommendation: log from persona consensus."
	case "model-chain-default-fallback":
		return "Make unknown model fallback deterministic and add regression coverage."
	default:
		return "Implement workstream changes with regression coverage."
	}
}

func writePRBody(issueID, workstream string) string {
	bodyPath := filepath.Join(".sdp", "pr-body-"+issueID+".md")
	body := "## Summary\n\n- worker workflow execution for " + issueID + "\n- implemented workstream: " + workstream + "\n"
	if err := os.WriteFile(bodyPath, []byte(body), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return bodyPath
}

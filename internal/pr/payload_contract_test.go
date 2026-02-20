package pr

import "testing"

func TestDefaultPRPayloadContractIncludesRequiredFields(t *testing.T) {
	contract := DefaultPRPayloadContract()
	if contract.ContractVersion != PRPayloadContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", contract.ContractVersion, PRPayloadContractVersion)
	}
	if len(contract.RequiredFields) == 0 {
		t.Fatal("expected required payload fields")
	}

	required := map[string]string{
		"event":                       "string",
		"issue.id":                    "string",
		"trace.run_id":                "string",
		"trace.pr_url":                "string",
		"trace.commit_ids":            "array[string]",
		"trace.run_context_link":      "string",
		"trace.evidence_context_link": "string",
		"pr.repository":               "string",
		"pr.base_branch":              "string",
		"pr.head_branch":              "string",
		"gates.pr_gate_passed":        "boolean",
		"gates.signals":               "array[object]",
		"published_at":                "string(datetime)",
	}

	seen := map[string]struct{}{}
	for _, field := range contract.RequiredFields {
		if field.Path == "" {
			t.Fatalf("field has empty path: %+v", field)
		}
		if field.Type == "" {
			t.Fatalf("field %s has empty type", field.Path)
		}
		if _, ok := seen[field.Path]; ok {
			t.Fatalf("duplicate field path: %s", field.Path)
		}
		seen[field.Path] = struct{}{}
	}

	for path, wantType := range required {
		found := false
		for _, field := range contract.RequiredFields {
			if field.Path == path {
				found = true
				if field.Type != wantType {
					t.Fatalf("unexpected field type for %s: got=%s want=%s", path, field.Type, wantType)
				}
				break
			}
		}
		if !found {
			t.Fatalf("missing required payload field: %s", path)
		}
	}
}

func TestDefaultCallbackChannelContractCoverage(t *testing.T) {
	contract := DefaultCallbackChannelContract()
	if contract.ContractVersion != CallbackChannelContractVersion {
		t.Fatalf("unexpected callback contract version: got=%s want=%s", contract.ContractVersion, CallbackChannelContractVersion)
	}
	if contract.Channel != "pr-callbacks.v1" {
		t.Fatalf("unexpected callback channel: %s", contract.Channel)
	}

	if len(contract.RequiredHeaders) == 0 {
		t.Fatal("expected callback headers")
	}
	headers := map[string]struct{}{}
	for _, header := range contract.RequiredHeaders {
		if header == "" {
			t.Fatal("callback header cannot be empty")
		}
		if _, ok := headers[header]; ok {
			t.Fatalf("duplicate callback header: %s", header)
		}
		headers[header] = struct{}{}
	}
	for _, required := range []string{"x-sdp-event", "x-sdp-issue-id", "x-sdp-run-id", "x-sdp-idempotency-key", "x-sdp-published-at"} {
		if _, ok := headers[required]; !ok {
			t.Fatalf("missing callback header: %s", required)
		}
	}

	recipients := map[string]CallbackRecipient{}
	for _, recipient := range contract.Recipients {
		if recipient.ID == "" {
			t.Fatalf("recipient has empty id: %+v", recipient)
		}
		if recipient.AddressSource == "" {
			t.Fatalf("recipient %s has empty address source", recipient.ID)
		}
		recipients[recipient.ID] = recipient
	}
	owner, ok := recipients["issue-owner"]
	if !ok {
		t.Fatal("missing issue-owner recipient")
	}
	if !owner.Required || !owner.AckRequired {
		t.Fatal("issue-owner must be required and acked")
	}
	audit, ok := recipients["orchestrator-audit"]
	if !ok {
		t.Fatal("missing orchestrator-audit recipient")
	}
	if !audit.Required || !audit.AckRequired {
		t.Fatal("orchestrator-audit must be required and acked")
	}

	guarantees := map[string]string{}
	for _, guarantee := range contract.DeliveryGuarantees {
		if guarantee.Name == "" || guarantee.Policy == "" {
			t.Fatalf("invalid guarantee: %+v", guarantee)
		}
		guarantees[guarantee.Name] = guarantee.Policy
	}
	if guarantees["delivery"] != "at-least-once" {
		t.Fatalf("unexpected delivery policy: %q", guarantees["delivery"])
	}
	if guarantees["idempotency"] != "required" {
		t.Fatalf("unexpected idempotency policy: %q", guarantees["idempotency"])
	}
	if guarantees["dead-letter"] != "required" {
		t.Fatalf("unexpected dead-letter policy: %q", guarantees["dead-letter"])
	}
}

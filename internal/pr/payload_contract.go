package pr

import "sort"

const (
	PRPayloadContractVersion       = "pr-payload/v1"
	CallbackChannelContractVersion = "callback-channel/v1"
)

type PayloadField struct {
	Path        string
	Type        string
	Description string
}

type CallbackRecipient struct {
	ID             string
	Description    string
	AddressSource  string
	Required       bool
	AckRequired    bool
	MaxAckLatencyS int
}

type DeliveryGuarantee struct {
	Name        string
	Policy      string
	Description string
}

type PRPayloadContract struct {
	ContractVersion string
	RequiredFields  []PayloadField
}

type CallbackChannelContract struct {
	ContractVersion    string
	Channel            string
	RequiredHeaders    []string
	Recipients         []CallbackRecipient
	DeliveryGuarantees []DeliveryGuarantee
}

var requiredPRPayloadFields = []PayloadField{
	{Path: "event", Type: "string", Description: "Notification kind; must be pr.published for this flow."},
	{Path: "issue.id", Type: "string", Description: "Beads issue identifier associated with the PR."},
	{Path: "trace.run_id", Type: "string", Description: "Deterministic run identifier for replay/idempotency."},
	{Path: "trace.pr_url", Type: "string", Description: "Published pull request URL."},
	{Path: "trace.commit_ids", Type: "array[string]", Description: "Commit lineage included in the published PR."},
	{Path: "trace.run_context_link", Type: "string", Description: "Run artifact context link for callback recipients."},
	{Path: "trace.evidence_context_link", Type: "string", Description: "Evidence artifact context link for callback recipients."},
	{Path: "pr.title", Type: "string", Description: "PR title visible to reviewers."},
	{Path: "pr.repository", Type: "string", Description: "Repository slug for callback routing."},
	{Path: "pr.base_branch", Type: "string", Description: "Merge target branch."},
	{Path: "pr.head_branch", Type: "string", Description: "Source branch published by automation."},
	{Path: "gates.pr_gate_passed", Type: "boolean", Description: "Publish gate decision used for callbacks."},
	{Path: "gates.signals", Type: "array[object]", Description: "Gate signal outcomes used for transition traceability."},
	{Path: "published_at", Type: "string(datetime)", Description: "UTC timestamp at callback publish time."},
}

var callbackRecipients = []CallbackRecipient{
	{
		ID:             "issue-owner",
		Description:    "Primary recipient mapped from issue owner metadata.",
		AddressSource:  "issue.owner",
		Required:       true,
		AckRequired:    true,
		MaxAckLatencyS: 30,
	},
	{
		ID:             "orchestrator-audit",
		Description:    "Operational audit sink for publish/callback traceability.",
		AddressSource:  "runtime.callback.audit_sink",
		Required:       true,
		AckRequired:    true,
		MaxAckLatencyS: 30,
	},
	{
		ID:             "watchers",
		Description:    "Optional subscriber list carried in issue metadata.",
		AddressSource:  "issue.callback_watchers",
		Required:       false,
		AckRequired:    false,
		MaxAckLatencyS: 0,
	},
}

var callbackDeliveryGuarantees = []DeliveryGuarantee{
	{Name: "delivery", Policy: "at-least-once", Description: "Publisher retries until acknowledged or dead-lettered."},
	{Name: "ordering", Policy: "per-issue-ordered", Description: "Callbacks for the same issue_id preserve publish order."},
	{Name: "idempotency", Policy: "required", Description: "idempotency_key header = issue.id + trace.run_id + trace.pr_url."},
	{Name: "retry-window", Policy: "max-15m", Description: "Retry budget is bounded to avoid unbounded backlog growth."},
	{Name: "dead-letter", Policy: "required", Description: "Expired deliveries move to callback dead-letter stream with reason."},
}

var callbackRequiredHeaders = []string{
	"x-sdp-event",
	"x-sdp-issue-id",
	"x-sdp-run-id",
	"x-sdp-idempotency-key",
	"x-sdp-published-at",
}

func DefaultPRPayloadContract() PRPayloadContract {
	fields := append([]PayloadField(nil), requiredPRPayloadFields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Path < fields[j].Path })
	return PRPayloadContract{
		ContractVersion: PRPayloadContractVersion,
		RequiredFields:  fields,
	}
}

func DefaultCallbackChannelContract() CallbackChannelContract {
	recipients := append([]CallbackRecipient(nil), callbackRecipients...)
	sort.Slice(recipients, func(i, j int) bool { return recipients[i].ID < recipients[j].ID })
	guarantees := append([]DeliveryGuarantee(nil), callbackDeliveryGuarantees...)
	sort.Slice(guarantees, func(i, j int) bool { return guarantees[i].Name < guarantees[j].Name })
	headers := append([]string(nil), callbackRequiredHeaders...)
	sort.Strings(headers)

	return CallbackChannelContract{
		ContractVersion:    CallbackChannelContractVersion,
		Channel:            "pr-callbacks.v1",
		RequiredHeaders:    headers,
		Recipients:         recipients,
		DeliveryGuarantees: guarantees,
	}
}

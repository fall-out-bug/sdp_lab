package pr

import "sort"

const (
	CallbackRoutingReliabilityContractVersion = "callback-routing-reliability/v1"
	UserNotificationPolicyContractVersion     = "user-notification-policy/v1"
)

type CallbackRetryStage struct {
	Attempt     int
	DelayS      int
	MaxJitterS  int
	Trigger     string
	Description string
}

type CallbackRoutingPolicyControl struct {
	Key           string
	Description   string
	DefaultValue  string
	AllowedValues []string
}

type CallbackRoutingReliabilityContract struct {
	ContractVersion    string
	AckTimeoutS        int
	MaxRetryWindowS    int
	RetryBudget        []CallbackRetryStage
	DeadLetterEnabled  bool
	DeadLetterReason   string
	RouteFallbackOrder []string
	PolicyControls     []CallbackRoutingPolicyControl
}

type UserNotificationPolicyContract struct {
	ContractVersion   string
	DefaultAudience   string
	RequiredAudiences []string
	EscalationMode    string
	PolicyControls    []CallbackRoutingPolicyControl
}

var callbackRetryBudget = []CallbackRetryStage{
	{Attempt: 1, DelayS: 5, MaxJitterS: 2, Trigger: "ack-timeout", Description: "Fast retry for transient receiver slowness."},
	{Attempt: 2, DelayS: 15, MaxJitterS: 4, Trigger: "retryable-status", Description: "Short backoff for temporary endpoint instability."},
	{Attempt: 3, DelayS: 30, MaxJitterS: 6, Trigger: "retryable-status", Description: "Escalate spacing to reduce callback burst pressure."},
	{Attempt: 4, DelayS: 60, MaxJitterS: 10, Trigger: "retryable-status", Description: "Medium interval retry while preserving delivery latency."},
	{Attempt: 5, DelayS: 120, MaxJitterS: 15, Trigger: "retryable-status", Description: "Long backoff as receiver instability persists."},
	{Attempt: 6, DelayS: 240, MaxJitterS: 20, Trigger: "retryable-status", Description: "Final long-gap retry before terminal attempt."},
	{Attempt: 7, DelayS: 420, MaxJitterS: 20, Trigger: "retryable-status", Description: "Terminal retry; failures dead-letter when budget expires."},
}

var callbackRoutingControls = []CallbackRoutingPolicyControl{
	{
		Key:           "callback.route.mode",
		Description:   "Chooses deterministic recipient routing strategy.",
		DefaultValue:  "required-first",
		AllowedValues: []string{"required-first", "fanout-all"},
	},
	{
		Key:           "callback.retry.profile",
		Description:   "Selects callback retry budget profile.",
		DefaultValue:  "standard-15m",
		AllowedValues: []string{"aggressive-5m", "standard-15m"},
	},
	{
		Key:           "callback.notify.watchers",
		Description:   "Controls optional watcher notifications.",
		DefaultValue:  "enabled",
		AllowedValues: []string{"enabled", "disabled"},
	},
	{
		Key:           "callback.escalate.on.deadletter",
		Description:   "Controls whether dead-lettered callbacks trigger user-visible escalation notice.",
		DefaultValue:  "enabled",
		AllowedValues: []string{"enabled", "disabled"},
	},
}

func DefaultCallbackRoutingReliabilityContract() CallbackRoutingReliabilityContract {
	budget := append([]CallbackRetryStage(nil), callbackRetryBudget...)
	sort.Slice(budget, func(i, j int) bool { return budget[i].Attempt < budget[j].Attempt })
	controls := append([]CallbackRoutingPolicyControl(nil), callbackRoutingControls...)
	sort.Slice(controls, func(i, j int) bool { return controls[i].Key < controls[j].Key })

	return CallbackRoutingReliabilityContract{
		ContractVersion:    CallbackRoutingReliabilityContractVersion,
		AckTimeoutS:        30,
		MaxRetryWindowS:    900,
		RetryBudget:        budget,
		DeadLetterEnabled:  true,
		DeadLetterReason:   "retry-window-exhausted",
		RouteFallbackOrder: []string{"issue-owner", "orchestrator-audit"},
		PolicyControls:     controls,
	}
}

func DefaultUserNotificationPolicyContract() UserNotificationPolicyContract {
	controls := append([]CallbackRoutingPolicyControl(nil), callbackRoutingControls...)
	sort.Slice(controls, func(i, j int) bool { return controls[i].Key < controls[j].Key })

	return UserNotificationPolicyContract{
		ContractVersion:   UserNotificationPolicyContractVersion,
		DefaultAudience:   "issue-owner",
		RequiredAudiences: []string{"issue-owner", "orchestrator-audit"},
		EscalationMode:    "dead-letter-notify",
		PolicyControls:    controls,
	}
}

func IsRetryableCallbackStatus(statusCode int) bool {
	if statusCode == 408 || statusCode == 425 || statusCode == 429 {
		return true
	}
	return statusCode >= 500 && statusCode <= 599
}

package modelgateway.routing

import future.keywords.if
import future.keywords.in

default allow = false
default provider = "openai"
default model = "gpt-4o-mini"

# Task class to model mapping
task_model_mapping := {
    "code": "gpt-4o",
    "analysis": "claude-3-sonnet",
    "creative": "claude-3-opus",
    "reasoning": "gpt-4-turbo",
    "embedding": "text-embedding-3-small"
}

# Sensitivity to provider restrictions
sensitivity_restrictions := {
    "public": ["openai", "anthropic", "selfhosted"],
    "internal": ["openai", "anthropic"],
    "confidential": ["selfhosted"],
    "restricted": ["selfhosted"]
}

# Provider latency estimates (ms)
provider_latency := {
    "openai": 500,
    "anthropic": 800,
    "selfhosted": 200
}

# Provider cost estimates (cents per 1k tokens)
provider_cost := {
    "openai": 3,
    "anthropic": 5,
    "selfhosted": 0
}

# Main routing decision
routing_decision := {
    "provider": selected_provider,
    "model": selected_model,
    "fallback_chain": fallback_chain,
    "reason": decision_reason
} if {
    input.task_class
    input.sensitivity
    
    selected_provider := get_provider_for_sensitivity(input.sensitivity)
    selected_model := get_model_for_task(input.task_class)
    fallback_chain := get_fallback_chain(selected_provider)
    decision_reason := get_decision_reason(selected_provider, input)
}

# Get allowed provider based on sensitivity
get_provider_for_sensitivity(sensitivity) := provider if {
    allowed := sensitivity_restrictions[sensitivity]
    provider := allowed[0]
}

# Get model based on task class
get_model_for_task(task_class) := model if {
    model := task_model_mapping[task_class]
}

get_model_for_task(task_class) := "gpt-4o-mini" if {
    not task_model_mapping[task_class]
}

# Build fallback chain excluding primary provider
get_fallback_chain(primary) := chain if {
    all_providers := ["openai", "anthropic", "selfhosted"]
    chain := [p | p := all_providers[_]; p != primary]
}

# Generate decision reason
get_decision_reason(provider, input) := reason if {
    reason := sprintf("selected %s for %s task with %s sensitivity", [provider, input.task_class, input.sensitivity])
}

# Allow if all constraints are satisfied
allow if {
    routing_decision.provider
    routing_decision.model
    
    # Check latency constraint
    satisfies_latency_constraint(input.max_latency_ms, routing_decision.provider)
    
    # Check cost constraint
    satisfies_cost_constraint(input.max_cost_cents, routing_decision.provider)
}

# Latency constraint check
satisfies_latency_constraint(max_ms, provider) if {
    max_ms > 0
    provider_latency[provider] <= max_ms
}

satisfies_latency_constraint(max_ms, provider) if {
    max_ms == 0
}

satisfies_latency_constraint(max_ms, provider) if {
    not max_ms
}

# Cost constraint check
satisfies_cost_constraint(max_cents, provider) if {
    max_cents > 0
    provider_cost[provider] <= max_cents
}

satisfies_cost_constraint(max_cents, provider) if {
    max_cents == 0
}

satisfies_cost_constraint(max_cents, provider) if {
    not max_cents
}

# Vision capability check
requires_vision_supported(provider) if {
    input.requires_vision == true
    provider == "openai"
}

requires_vision_supported(provider) if {
    input.requires_vision == true
    provider == "anthropic"
}

requires_vision_supported(provider) if {
    not input.requires_vision
}

requires_vision_supported(provider) if {
    input.requires_vision == false
}

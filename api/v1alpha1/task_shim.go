package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskPhase mirrors kubeopencode Task status.phase.
// Compatible with adapter.CRDPhase (Pending, Running, Succeeded, Completed, Failed).
type TaskPhase string

const (
	TaskPhasePending   TaskPhase = "Pending"
	TaskPhaseRunning    TaskPhase = "Running"
	TaskPhaseSucceeded  TaskPhase = "Succeeded"
	TaskPhaseCompleted  TaskPhase = "Completed"
	TaskPhaseFailed     TaskPhase = "Failed"
)

// +kubebuilder:object:generate=true
// TerminalReason mirrors kubeopencode Task status.terminalReason.
type TerminalReason struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Task is a shim for kubeopencode Task CRD.
// Provides interface-compatible types until upstream types are available.
// Compatible with internal/adapter (CRDPhase maps to TaskPhase).
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
// TaskSpec defines the desired state of Task.
type TaskSpec struct {
	// AgentRef identifies the agent/model for this task.
	AgentRef AgentRef `json:"agentRef,omitempty"`
	// Prompt is the task prompt.
	Prompt string `json:"prompt,omitempty"`
	// Objective is the acceptance criteria.
	Objective string `json:"objective,omitempty"`
	// DependsOn lists Task names that must reach terminal phase before this Task runs.
	// Fork-first (WS-021-01): kubeopencode controller honors this for DAG ordering.
	DependsOn []string `json:"dependsOn,omitempty"`
}

// +kubebuilder:object:generate=true
// AgentRef references an agent or model.
type AgentRef struct {
	Model string `json:"model,omitempty"`
}

// +kubebuilder:object:generate=true
// TaskStatus defines the observed state of Task.
type TaskStatus struct {
	Phase          TaskPhase     `json:"phase,omitempty"`
	TerminalReason *TerminalReason `json:"terminalReason,omitempty"`
}

// +kubebuilder:object:root=true

// TaskList contains a list of Task.
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

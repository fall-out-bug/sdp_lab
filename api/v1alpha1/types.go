package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Issue",type=string,JSONPath=`.spec.issueId`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AgentRun is the Schema for the agentruns API.
type AgentRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentRunSpec   `json:"spec,omitempty"`
	Status AgentRunStatus `json:"status,omitempty"`
}

// AgentRunSpec defines the desired state of AgentRun.
type AgentRunSpec struct {
	IssueID    string `json:"issueId"`
	Repo       string `json:"repo"`
	BaseBranch string `json:"baseBranch"`
	Model      string `json:"model"`
	Workstream string `json:"workstream"`
	TimeoutSec int    `json:"timeoutSec,omitempty"`
}

// +kubebuilder:object:generate=true
// AgentRunStatus defines the observed state of AgentRun.
type AgentRunStatus struct {
	Phase         string             `json:"phase"`
	Conditions    []metav1.Condition `json:"conditions,omitempty"`
	WorkerTask    string             `json:"workerTask,omitempty"`
	ReviewerTask  string             `json:"reviewerTask,omitempty"`
	PrURL         string             `json:"prUrl,omitempty"`
	LastError     string             `json:"lastError,omitempty"`
}

// +kubebuilder:object:root=true

// AgentRunList contains a list of AgentRun.
type AgentRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentRun `json:"items"`
}

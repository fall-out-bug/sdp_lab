// Package v1alpha1 contains API types for the SDP adapter controller.
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is the group version for SDP API types.
	GroupVersion = schema.GroupVersion{Group: "sdp.dev", Version: "v1alpha1"}

	// SchemeBuilder is used to add types to the scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds all types in this group version to the scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&AgentRun{}, &AgentRunList{})
	SchemeBuilder.Register(&Task{}, &TaskList{})
}

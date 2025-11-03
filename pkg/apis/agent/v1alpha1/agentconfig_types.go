package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Max Iterations",type=integer,JSONPath=`.spec.maxIterations`
// +kubebuilder:printcolumn:name="Service Account",type=string,JSONPath=`.spec.serviceAccount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AgentConfig defines the configuration for agent execution
type AgentConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec AgentConfigSpec `json:"spec,omitempty"`

	// +optional
	Status AgentConfigStatus `json:"status,omitempty"`
}

// AgentConfigSpec defines the desired state of AgentConfig
type AgentConfigSpec struct {
	// ServiceAccount to use for agent pod execution
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// ConfigPVC is the name of the PVC containing prompts, schemas, and policies
	// +kubebuilder:validation:MinLength=1
	ConfigPVC string `json:"configPVC"`

	// MaxIterations is the maximum number of plan-act-reflect iterations
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	MaxIterations int32 `json:"maxIterations,omitempty"`

	// Timeout is the maximum duration for agent execution
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// PreHooks are the names of hook pods to run before agent execution
	// +optional
	// +listType=atomic
	PreHooks []string `json:"preHooks,omitempty"`

	// PostHooks are the names of hook pods to run after agent execution
	// +optional
	// +listType=atomic
	PostHooks []string `json:"postHooks,omitempty"`

	// Policy defines the OPA policy enforcement mode
	// +optional
	Policy PolicySpec `json:"policy,omitempty"`

	// NetworkPolicy defines the network isolation mode
	// +optional
	// +kubebuilder:validation:Enum=strict;permissive
	NetworkPolicy string `json:"networkPolicy,omitempty"`

	// Provider specifies which LLM provider to use
	// +optional
	// +kubebuilder:validation:Enum=claude;gemini
	Provider string `json:"provider,omitempty"`
}

// PolicySpec defines OPA policy configuration
type PolicySpec struct {
	// OPA defines the policy enforcement mode
	// +optional
	// +kubebuilder:validation:Enum=strict;permissive
	OPA string `json:"opa,omitempty"`
}

// AgentConfigStatus defines the observed state of AgentConfig
type AgentConfigStatus struct {
	// Conditions represent the latest available observations of the AgentConfig's state
	// +optional
	// +listType=atomic
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AgentConfigList contains a list of AgentConfig
type AgentConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentConfig `json:"items"`
}

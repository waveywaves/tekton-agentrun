package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Iterations",type=integer,JSONPath=`.status.iterations`
// +kubebuilder:printcolumn:name="Started",type=date,JSONPath=`.status.startTime`
// +kubebuilder:printcolumn:name="Completed",type=date,JSONPath=`.status.completionTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AgentRun represents a single execution of an agent
type AgentRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec AgentRunSpec `json:"spec,omitempty"`

	// +optional
	Status AgentRunStatus `json:"status,omitempty"`
}

// AgentRunSpec defines the desired state of AgentRun
type AgentRunSpec struct {
	// ConfigRef references the AgentConfig to use
	ConfigRef ConfigRef `json:"configRef"`

	// Goal is the objective for the agent to achieve
	// +kubebuilder:validation:MinLength=1
	Goal string `json:"goal"`

	// Context provides additional information for the agent
	// +optional
	Context AgentContext `json:"context,omitempty"`
}

// ConfigRef references an AgentConfig
type ConfigRef struct {
	// Name of the AgentConfig
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// AgentContext provides additional context for agent execution
type AgentContext struct {
	// Hints provide guidance for the agent
	// +optional
	// +listType=atomic
	Hints []string `json:"hints,omitempty"`
}

// AgentRunStatus defines the observed state of AgentRun
type AgentRunStatus struct {
	// Conditions represent the latest available observations of the AgentRun's state
	// +optional
	// +listType=atomic
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of execution
	// +optional
	// +kubebuilder:validation:Enum=Pending;PreHooks;Planning;Acting;Reflecting;PostHooks;Succeeded;Failed
	Phase string `json:"phase,omitempty"`

	// StartTime is when the AgentRun started executing
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the AgentRun completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Iterations is the number of plan-act-reflect iterations completed
	// +optional
	Iterations int32 `json:"iterations,omitempty"`

	// Results contains the output from the agent
	// +optional
	// +listType=atomic
	Results []AgentResult `json:"results,omitempty"`
}

// AgentResult represents a result from the agent
type AgentResult struct {
	// Name of the result
	Name string `json:"name"`

	// Value of the result
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AgentRunList contains a list of AgentRun
type AgentRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentRun `json:"items"`
}

// Phase constants
const (
	AgentRunPhasePending    = "Pending"
	AgentRunPhasePreHooks   = "PreHooks"
	AgentRunPhasePlanning   = "Planning"
	AgentRunPhaseActing     = "Acting"
	AgentRunPhaseReflecting = "Reflecting"
	AgentRunPhasePostHooks  = "PostHooks"
	AgentRunPhaseSucceeded  = "Succeeded"
	AgentRunPhaseFailed     = "Failed"
)

// IsDone returns true if the AgentRun has completed (succeeded or failed)
func (ar *AgentRun) IsDone() bool {
	return ar.Status.Phase == AgentRunPhaseSucceeded || ar.Status.Phase == AgentRunPhaseFailed
}

// HasStarted returns true if the AgentRun has started
func (ar *AgentRun) HasStarted() bool {
	return ar.Status.StartTime != nil
}

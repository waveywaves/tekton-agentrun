package v1alpha1

import (
	"context"
)

// SetDefaults sets default values for AgentRun
func (ar *AgentRun) SetDefaults(ctx context.Context) {
	ar.Spec.SetDefaults(ctx)

	// Initialize status phase if not set
	if ar.Status.Phase == "" {
		ar.Status.Phase = AgentRunPhasePending
	}
}

// SetDefaults sets default values for AgentRunSpec
func (ars *AgentRunSpec) SetDefaults(ctx context.Context) {
	// AgentRunSpec has no fields that need defaults currently
	// Context and hints are optional and have no defaults
}

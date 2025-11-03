package v1alpha1

import (
	"context"
	"fmt"
)

// Validate validates the AgentRun
func (ar *AgentRun) Validate(ctx context.Context) error {
	if err := validateObjectMeta(ar.ObjectMeta); err != nil {
		return fmt.Errorf("metadata: %w", err)
	}

	if ar.ObjectMeta.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	return ar.Spec.Validate(ctx)
}

// Validate validates the AgentRunSpec
func (ars *AgentRunSpec) Validate(ctx context.Context) error {
	if ars.ConfigRef.Name == "" {
		return fmt.Errorf("configRef.name is required")
	}

	if ars.Goal == "" {
		return fmt.Errorf("goal is required")
	}

	return nil
}

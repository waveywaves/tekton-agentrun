package v1alpha1

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Validate validates the AgentConfig
func (ac *AgentConfig) Validate(ctx context.Context) error {
	if err := validateObjectMeta(ac.ObjectMeta); err != nil {
		return fmt.Errorf("metadata: %w", err)
	}
	return ac.Spec.Validate(ctx)
}

// Validate validates the AgentConfigSpec
func (acs *AgentConfigSpec) Validate(ctx context.Context) error {
	if acs.ConfigPVC == "" {
		return fmt.Errorf("configPVC is required")
	}

	// MaxIterations is optional, but if set must be in range 1-10
	if acs.MaxIterations < 0 || acs.MaxIterations > 10 {
		return fmt.Errorf("maxIterations must be between 0 and 10")
	}

	if acs.Provider != "" && acs.Provider != "claude" && acs.Provider != "gemini" {
		return fmt.Errorf("provider must be either 'claude' or 'gemini'")
	}

	if acs.NetworkPolicy != "" && acs.NetworkPolicy != "strict" && acs.NetworkPolicy != "permissive" {
		return fmt.Errorf("networkPolicy must be either 'strict' or 'permissive'")
	}

	if acs.Policy.OPA != "" && acs.Policy.OPA != "strict" && acs.Policy.OPA != "permissive" {
		return fmt.Errorf("policy.opa must be either 'strict' or 'permissive'")
	}

	return nil
}

// validateObjectMeta validates object metadata
func validateObjectMeta(meta metav1.ObjectMeta) error {
	if meta.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(meta.Name) > 253 {
		return fmt.Errorf("name is too long (max 253 characters)")
	}
	return nil
}

package v1alpha1

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultMaxIterations  = 3
	DefaultTimeout        = 8 * time.Minute
	DefaultProvider       = "claude"
	DefaultNetworkPolicy  = "strict"
	DefaultOPAPolicy      = "strict"
	DefaultServiceAccount = "default"
)

// SetDefaults sets default values for AgentConfig
func (ac *AgentConfig) SetDefaults(ctx context.Context) {
	ac.Spec.SetDefaults(ctx)
}

// SetDefaults sets default values for AgentConfigSpec
func (acs *AgentConfigSpec) SetDefaults(ctx context.Context) {
	if acs.ServiceAccount == "" {
		acs.ServiceAccount = DefaultServiceAccount
	}

	if acs.MaxIterations == 0 {
		acs.MaxIterations = DefaultMaxIterations
	}

	if acs.Timeout == nil {
		acs.Timeout = &metav1.Duration{Duration: DefaultTimeout}
	}

	if acs.Provider == "" {
		acs.Provider = DefaultProvider
	}

	if acs.NetworkPolicy == "" {
		acs.NetworkPolicy = DefaultNetworkPolicy
	}

	if acs.Policy.OPA == "" {
		acs.Policy.OPA = DefaultOPAPolicy
	}
}

package v1alpha1

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAgentConfigSpec_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		spec     *AgentConfigSpec
		expected *AgentConfigSpec
	}{
		{
			name: "all fields empty",
			spec: &AgentConfigSpec{
				ConfigPVC: "test-pvc",
			},
			expected: &AgentConfigSpec{
				ConfigPVC:      "test-pvc",
				ServiceAccount: DefaultServiceAccount,
				MaxIterations:  DefaultMaxIterations,
				Timeout:        &metav1.Duration{Duration: DefaultTimeout},
				Provider:       DefaultProvider,
				NetworkPolicy:  DefaultNetworkPolicy,
				Policy: PolicySpec{
					OPA: DefaultOPAPolicy,
				},
			},
		},
		{
			name: "some fields set",
			spec: &AgentConfigSpec{
				ConfigPVC:     "test-pvc",
				Provider:      "gemini",
				MaxIterations: 5,
			},
			expected: &AgentConfigSpec{
				ConfigPVC:      "test-pvc",
				ServiceAccount: DefaultServiceAccount,
				MaxIterations:  5, // Should keep existing value
				Timeout:        &metav1.Duration{Duration: DefaultTimeout},
				Provider:       "gemini", // Should keep existing value
				NetworkPolicy:  DefaultNetworkPolicy,
				Policy: PolicySpec{
					OPA: DefaultOPAPolicy,
				},
			},
		},
		{
			name: "all fields set",
			spec: &AgentConfigSpec{
				ConfigPVC:      "test-pvc",
				ServiceAccount: "custom-sa",
				MaxIterations:  7,
				Timeout:        &metav1.Duration{Duration: 10 * time.Minute},
				Provider:       "claude",
				NetworkPolicy:  "permissive",
				Policy: PolicySpec{
					OPA: "permissive",
				},
			},
			expected: &AgentConfigSpec{
				ConfigPVC:      "test-pvc",
				ServiceAccount: "custom-sa",
				MaxIterations:  7,
				Timeout:        &metav1.Duration{Duration: 10 * time.Minute},
				Provider:       "claude",
				NetworkPolicy:  "permissive",
				Policy: PolicySpec{
					OPA: "permissive",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.spec.SetDefaults(context.Background())

			if tt.spec.ServiceAccount != tt.expected.ServiceAccount {
				t.Errorf("ServiceAccount = %v, want %v", tt.spec.ServiceAccount, tt.expected.ServiceAccount)
			}
			if tt.spec.MaxIterations != tt.expected.MaxIterations {
				t.Errorf("MaxIterations = %v, want %v", tt.spec.MaxIterations, tt.expected.MaxIterations)
			}
			if tt.spec.Timeout.Duration != tt.expected.Timeout.Duration {
				t.Errorf("Timeout = %v, want %v", tt.spec.Timeout.Duration, tt.expected.Timeout.Duration)
			}
			if tt.spec.Provider != tt.expected.Provider {
				t.Errorf("Provider = %v, want %v", tt.spec.Provider, tt.expected.Provider)
			}
			if tt.spec.NetworkPolicy != tt.expected.NetworkPolicy {
				t.Errorf("NetworkPolicy = %v, want %v", tt.spec.NetworkPolicy, tt.expected.NetworkPolicy)
			}
			if tt.spec.Policy.OPA != tt.expected.Policy.OPA {
				t.Errorf("Policy.OPA = %v, want %v", tt.spec.Policy.OPA, tt.expected.Policy.OPA)
			}
		})
	}
}

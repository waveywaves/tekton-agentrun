package v1alpha1

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAgentConfigSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *AgentConfigSpec
		wantErr bool
	}{
		{
			name: "valid spec",
			spec: &AgentConfigSpec{
				ConfigPVC:     "agent-config",
				MaxIterations: 3,
			},
			wantErr: false,
		},
		{
			name: "missing configPVC",
			spec: &AgentConfigSpec{
				MaxIterations: 3,
			},
			wantErr: true,
		},
		{
			name: "empty configPVC",
			spec: &AgentConfigSpec{
				ConfigPVC:     "",
				MaxIterations: 3,
			},
			wantErr: true,
		},
		{
			name: "maxIterations too high",
			spec: &AgentConfigSpec{
				ConfigPVC:     "agent-config",
				MaxIterations: 11,
			},
			wantErr: true,
		},
		{
			name: "maxIterations zero - valid (default)",
			spec: &AgentConfigSpec{
				ConfigPVC:     "agent-config",
				MaxIterations: 0,
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			spec: &AgentConfigSpec{
				ConfigPVC: "agent-config",
				Provider:  "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid provider claude",
			spec: &AgentConfigSpec{
				ConfigPVC: "agent-config",
				Provider:  "claude",
			},
			wantErr: false,
		},
		{
			name: "valid provider gemini",
			spec: &AgentConfigSpec{
				ConfigPVC: "agent-config",
				Provider:  "gemini",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentConfigSpec.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AgentConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AgentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-config",
				},
				Spec: AgentConfigSpec{
					ConfigPVC:     "agent-config",
					MaxIterations: 3,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid name - too long",
			config: &AgentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "this-is-a-very-long-name-that-exceeds-the-maximum-allowed-length-for-kubernetes-resource-names-and-it-keeps-going-and-going-and-going-until-it-reaches-well-over-two-hundred-and-fifty-three-characters-which-is-the-maximum-allowed-by-kubernetes-for-resource-names-so-this-should-definitely-fail-validation",
				},
				Spec: AgentConfigSpec{
					ConfigPVC: "agent-config",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

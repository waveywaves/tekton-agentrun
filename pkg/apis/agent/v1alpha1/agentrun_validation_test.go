package v1alpha1

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAgentRunSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *AgentRunSpec
		wantErr bool
	}{
		{
			name: "valid spec",
			spec: &AgentRunSpec{
				ConfigRef: ConfigRef{
					Name: "test-config",
				},
				Goal: "Debug deployment failures",
			},
			wantErr: false,
		},
		{
			name: "missing configRef name",
			spec: &AgentRunSpec{
				ConfigRef: ConfigRef{
					Name: "",
				},
				Goal: "Debug deployment failures",
			},
			wantErr: true,
		},
		{
			name: "missing goal",
			spec: &AgentRunSpec{
				ConfigRef: ConfigRef{
					Name: "test-config",
				},
				Goal: "",
			},
			wantErr: true,
		},
		{
			name: "valid with context hints",
			spec: &AgentRunSpec{
				ConfigRef: ConfigRef{
					Name: "test-config",
				},
				Goal: "Debug deployment failures",
				Context: AgentContext{
					Hints: []string{"Check events", "Review logs"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentRunSpec.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentRun_Validate(t *testing.T) {
	tests := []struct {
		name    string
		run     *AgentRun
		wantErr bool
	}{
		{
			name: "valid run",
			run: &AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: AgentRunSpec{
					ConfigRef: ConfigRef{
						Name: "test-config",
					},
					Goal: "Debug deployment failures",
				},
			},
			wantErr: false,
		},
		{
			name: "missing namespace",
			run: &AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-run",
				},
				Spec: AgentRunSpec{
					ConfigRef: ConfigRef{
						Name: "test-config",
					},
					Goal: "Debug deployment failures",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run.Validate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentRun.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

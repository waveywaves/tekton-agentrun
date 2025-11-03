package agent

import (
	"context"
	"testing"
)

func TestOPAPolicy_Allow(t *testing.T) {
	// Create a simple policy that allows k8s_get_resources in specific namespaces
	policyContent := `
package agent.tools

default allow = false

allow {
    input.tool == "k8s_get_resources"
    data.allowed_namespaces[_] == input.namespace
}

allow {
    input.tool == "k8s_get_logs"
    data.allowed_namespaces[_] == input.namespace
}
`

	policy := &OPAPolicy{
		PolicyContent: policyContent,
		Data: map[string]interface{}{
			"allowed_namespaces": []string{"default", "kube-system"},
		},
	}

	err := policy.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	tests := []struct {
		name      string
		toolCall  ToolCall
		wantAllow bool
	}{
		{
			name: "allowed tool in allowed namespace",
			toolCall: ToolCall{
				Name: "k8s_get_resources",
				Input: map[string]interface{}{
					"namespace": "default",
				},
			},
			wantAllow: true,
		},
		{
			name: "allowed tool in disallowed namespace",
			toolCall: ToolCall{
				Name: "k8s_get_resources",
				Input: map[string]interface{}{
					"namespace": "production",
				},
			},
			wantAllow: false,
		},
		{
			name: "get_logs in allowed namespace",
			toolCall: ToolCall{
				Name: "k8s_get_logs",
				Input: map[string]interface{}{
					"namespace": "kube-system",
				},
			},
			wantAllow: true,
		},
		{
			name: "unknown tool",
			toolCall: ToolCall{
				Name: "unknown_tool",
				Input: map[string]interface{}{
					"namespace": "default",
				},
			},
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := policy.Allow(ctx, tt.toolCall)

			gotAllow := err == nil

			if gotAllow != tt.wantAllow {
				t.Errorf("Allow() = %v (err=%v), want %v", gotAllow, err, tt.wantAllow)
			}
		})
	}
}

func TestOPAPolicy_FailClosed(t *testing.T) {
	// Invalid policy should fail closed
	policy := &OPAPolicy{
		PolicyContent: "invalid rego syntax {{{",
	}

	err := policy.Initialize()
	if err == nil {
		t.Fatal("Expected error for invalid policy, got nil")
	}
}

func TestOPAPolicy_EmptyPolicy(t *testing.T) {
	// Empty policy should deny everything
	policy := &OPAPolicy{
		PolicyContent: `
package agent.tools

default allow = false
`,
	}

	err := policy.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	ctx := context.Background()
	err = policy.Allow(ctx, ToolCall{
		Name: "k8s_get_resources",
		Input: map[string]interface{}{
			"namespace": "default",
		},
	})

	if err == nil {
		t.Error("Expected error (deny) for empty policy, got nil")
	}
}

func TestOPAPolicy_ComplexRules(t *testing.T) {
	// Policy with multiple conditions
	policyContent := `
package agent.tools

default allow = false

# Allow get_resources with label selector
allow {
    input.tool == "k8s_get_resources"
    input.namespace == "default"
    input.labelSelector != ""
}

# Deny if trying to list all pods without selector
deny[msg] {
    input.tool == "k8s_get_resources"
    input.resourceType == "pods"
    not input.labelSelector
    msg := "label selector required for pod listing"
}
`

	policy := &OPAPolicy{
		PolicyContent: policyContent,
	}

	err := policy.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	tests := []struct {
		name      string
		toolCall  ToolCall
		wantAllow bool
	}{
		{
			name: "with label selector",
			toolCall: ToolCall{
				Name: "k8s_get_resources",
				Input: map[string]interface{}{
					"namespace":     "default",
					"resourceType":  "pods",
					"labelSelector": "app=test",
				},
			},
			wantAllow: true,
		},
		{
			name: "without label selector",
			toolCall: ToolCall{
				Name: "k8s_get_resources",
				Input: map[string]interface{}{
					"namespace":    "default",
					"resourceType": "pods",
				},
			},
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := policy.Allow(ctx, tt.toolCall)

			gotAllow := err == nil

			if gotAllow != tt.wantAllow {
				t.Errorf("Allow() = %v (err=%v), want %v", gotAllow, err, tt.wantAllow)
			}
		})
	}
}

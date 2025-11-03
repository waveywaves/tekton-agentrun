package agent

import (
	"context"
	"errors"
	"testing"
)

// mockProvider implements Provider for testing
type mockProvider struct {
	responses []*Response
	callCount int
}

func (m *mockProvider) Call(ctx context.Context, messages []Message) (*Response, error) {
	if m.callCount >= len(m.responses) {
		return nil, errors.New("no more responses")
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

// mockTool implements Tool for testing
type mockTool struct {
	name   string
	result string
	err    error
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.result, nil
}

// mockPolicy for testing
type mockPolicy struct {
	allowAll bool
}

func (m *mockPolicy) Allow(ctx context.Context, toolCall ToolCall) error {
	if m.allowAll {
		return nil
	}
	return errors.New("policy violation")
}

func TestLoop_SuccessfulCompletion(t *testing.T) {
	// Setup mock provider that returns confident response on first iteration
	provider := &mockProvider{
		responses: []*Response{
			{
				Content:    "I've analyzed the goal and here's the plan",
				ToolCalls:  []ToolCall{{ID: "1", Name: "k8s_get_resources", Input: map[string]interface{}{"namespace": "default", "resourceType": "pods"}}},
				StopReason: "tool_use",
				TokensIn:   100,
				TokensOut:  50,
			},
			{
				Content:    "Based on the results, I'm confident the goal is achieved",
				StopReason: "end_turn",
				TokensIn:   200,
				TokensOut:  30,
			},
		},
	}

	tool := &mockTool{
		name:   "k8s_get_resources",
		result: `{"pods": [{"name": "test-pod"}]}`,
	}

	policy := &mockPolicy{allowAll: true}

	loop := &Loop{
		Provider:      provider,
		Tools:         map[string]Tool{"k8s_get_resources": tool},
		Policy:        policy,
		Goal:          "Check pods in default namespace",
		MaxIterations: 3,
	}

	ctx := context.Background()
	result, err := loop.Run(ctx)

	if err != nil {
		t.Fatalf("Loop.Run() error = %v, want nil", err)
	}

	if result.Status != "succeeded" {
		t.Errorf("Result.Status = %v, want succeeded", result.Status)
	}

	if result.Iterations != 1 {
		t.Errorf("Result.Iterations = %d, want 1", result.Iterations)
	}

	if len(result.ToolCalls) != 1 {
		t.Errorf("Result.ToolCalls length = %d, want 1", len(result.ToolCalls))
	}

	if result.TotalTokensIn != 300 {
		t.Errorf("Result.TotalTokensIn = %d, want 300", result.TotalTokensIn)
	}
}

func TestLoop_MaxIterationsReached(t *testing.T) {
	// Setup mock provider that never returns confident response
	provider := &mockProvider{
		responses: []*Response{
			{Content: "Iteration 1", StopReason: "end_turn", TokensIn: 100, TokensOut: 50},
			{Content: "Iteration 2", StopReason: "end_turn", TokensIn: 100, TokensOut: 50},
			{Content: "Iteration 3", StopReason: "end_turn", TokensIn: 100, TokensOut: 50},
		},
	}

	policy := &mockPolicy{allowAll: true}

	loop := &Loop{
		Provider:      provider,
		Tools:         map[string]Tool{},
		Policy:        policy,
		Goal:          "Test goal",
		MaxIterations: 3,
	}

	ctx := context.Background()
	result, err := loop.Run(ctx)

	if err != nil {
		t.Fatalf("Loop.Run() error = %v, want nil", err)
	}

	if result.Status != "max_iterations" {
		t.Errorf("Result.Status = %v, want max_iterations", result.Status)
	}

	if result.Iterations != 3 {
		t.Errorf("Result.Iterations = %d, want 3", result.Iterations)
	}
}

func TestLoop_PolicyViolation(t *testing.T) {
	provider := &mockProvider{
		responses: []*Response{
			{
				Content:    "Let me check the pods",
				ToolCalls:  []ToolCall{{ID: "1", Name: "k8s_get_resources", Input: map[string]interface{}{}}},
				StopReason: "tool_use",
				TokensIn:   100,
				TokensOut:  50,
			},
		},
	}

	tool := &mockTool{
		name:   "k8s_get_resources",
		result: "should not get here",
	}

	policy := &mockPolicy{allowAll: false} // Deny all

	loop := &Loop{
		Provider:      provider,
		Tools:         map[string]Tool{"k8s_get_resources": tool},
		Policy:        policy,
		Goal:          "Test goal",
		MaxIterations: 3,
	}

	ctx := context.Background()
	result, err := loop.Run(ctx)

	if err == nil {
		t.Fatal("Loop.Run() error = nil, want policy violation error")
	}

	if result.Status != "failed" {
		t.Errorf("Result.Status = %v, want failed", result.Status)
	}

	if len(result.ToolCalls) != 0 {
		t.Errorf("Result.ToolCalls length = %d, want 0 (should fail before executing)", len(result.ToolCalls))
	}
}

func TestLoop_ToolNotFound(t *testing.T) {
	provider := &mockProvider{
		responses: []*Response{
			{
				Content:    "Let me use unknown tool",
				ToolCalls:  []ToolCall{{ID: "1", Name: "unknown_tool", Input: map[string]interface{}{}}},
				StopReason: "tool_use",
				TokensIn:   100,
				TokensOut:  50,
			},
		},
	}

	policy := &mockPolicy{allowAll: true}

	loop := &Loop{
		Provider:      provider,
		Tools:         map[string]Tool{}, // No tools registered
		Policy:        policy,
		Goal:          "Test goal",
		MaxIterations: 3,
	}

	ctx := context.Background()
	result, err := loop.Run(ctx)

	if err == nil {
		t.Fatal("Loop.Run() error = nil, want tool not found error")
	}

	if result.Status != "failed" {
		t.Errorf("Result.Status = %v, want failed", result.Status)
	}
}

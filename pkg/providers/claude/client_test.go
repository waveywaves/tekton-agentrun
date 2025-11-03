package claude

import (
	"context"
	"os"
	"testing"

	"github.com/waveywaves/agentrun-controller/pkg/agent"
)

func TestClient_Call(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("CLAUDE_API_KEY not set, skipping integration test")
	}

	client := &Client{
		APIKey:      apiKey,
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   1024,
		Temperature: 0.2,
		TopP:        0.3,
	}

	ctx := context.Background()
	messages := []agent.Message{
		{
			Role:    "user",
			Content: "What is 2+2? Answer with just the number.",
		},
	}

	response, err := client.Call(ctx, messages)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if response.Content == "" {
		t.Error("Response content is empty")
	}

	if response.TokensIn == 0 {
		t.Error("TokensIn should be > 0")
	}

	if response.TokensOut == 0 {
		t.Error("TokensOut should be > 0")
	}

	t.Logf("Response: %s", response.Content)
	t.Logf("Tokens: in=%d, out=%d", response.TokensIn, response.TokensOut)
}

func TestClient_CallWithTools(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("CLAUDE_API_KEY not set, skipping integration test")
	}

	client := &Client{
		APIKey:      apiKey,
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   1024,
		Temperature: 0.2,
		TopP:        0.3,
	}

	// Define a simple tool
	tools := []Tool{
		{
			Name:        "get_weather",
			Description: "Get the weather for a location",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city name",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	client.Tools = tools

	ctx := context.Background()
	messages := []agent.Message{
		{
			Role:    "user",
			Content: "What's the weather in San Francisco?",
		},
	}

	response, err := client.Call(ctx, messages)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Should have tool calls since we asked about weather
	if len(response.ToolCalls) == 0 {
		t.Log("Warning: Expected tool calls but got none. Response:", response.Content)
		// Don't fail - Claude might choose not to use tools
	} else {
		t.Logf("Tool calls: %d", len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			t.Logf("Tool call %d: %s with input %v", i, tc.Name, tc.Input)
		}
	}
}

func TestClient_InvalidAPIKey(t *testing.T) {
	client := &Client{
		APIKey:      "invalid-key",
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   1024,
		Temperature: 0.2,
		TopP:        0.3,
	}

	ctx := context.Background()
	messages := []agent.Message{
		{
			Role:    "user",
			Content: "Hello",
		},
	}

	_, err := client.Call(ctx, messages)
	if err == nil {
		t.Fatal("Expected error with invalid API key, got nil")
	}

	t.Logf("Got expected error: %v", err)
}

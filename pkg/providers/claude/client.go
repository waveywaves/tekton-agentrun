package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/waveywaves/agentrun-controller/pkg/agent"
)

const (
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
)

// Client implements the agent.Provider interface for Claude
type Client struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	TopP        float64
	Tools       []Tool
	HTTPClient  *http.Client
}

// Tool represents a Claude tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// messagesRequest is the request body for the Messages API
type messagesRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Messages    []claudeMessage `json:"messages"`
	Tools       []Tool          `json:"tools,omitempty"`
	System      string          `json:"system,omitempty"`
}

// claudeMessage represents a message in Claude's format
type claudeMessage struct {
	Role    string        `json:"role"`
	Content []contentBlock `json:"content"`
}

// contentBlock can be text or tool use/result
type contentBlock struct {
	Type      string                 `json:"type"` // "text", "tool_use", "tool_result"
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Content   string                 `json:"content,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	IsError   bool                   `json:"is_error,omitempty"`
}

// messagesResponse is the response from the Messages API
type messagesResponse struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Content      []contentBlock  `json:"content"`
	Model        string          `json:"model"`
	StopReason   string          `json:"stop_reason"`
	StopSequence string          `json:"stop_sequence,omitempty"`
	Usage        usageInfo       `json:"usage"`
}

// usageInfo contains token usage information
type usageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Call implements agent.Provider.Call
func (c *Client) Call(ctx context.Context, messages []agent.Message) (*agent.Response, error) {
	// Convert messages to Claude format
	claudeMessages, systemPrompt := c.convertMessages(messages)

	// Build request
	reqBody := messagesRequest{
		Model:       c.Model,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
		TopP:        c.TopP,
		Messages:    claudeMessages,
		System:      systemPrompt,
	}

	if len(c.Tools) > 0 {
		reqBody.Tools = c.Tools
	}

	// Marshal request
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	// Get HTTP client
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	// Send request
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp messagesResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to agent.Response
	return c.convertResponse(&apiResp), nil
}

// convertMessages converts agent.Message to Claude format
func (c *Client) convertMessages(messages []agent.Message) ([]claudeMessage, string) {
	var claudeMessages []claudeMessage
	var systemPrompt string

	for _, msg := range messages {
		if msg.Role == "system" {
			// System messages are passed separately in Claude API
			systemPrompt = msg.Content
			continue
		}

		claudeMessages = append(claudeMessages, claudeMessage{
			Role: msg.Role,
			Content: []contentBlock{
				{
					Type: "text",
					Text: msg.Content,
				},
			},
		})
	}

	return claudeMessages, systemPrompt
}

// convertResponse converts Claude response to agent.Response
func (c *Client) convertResponse(resp *messagesResponse) *agent.Response {
	response := &agent.Response{
		StopReason: resp.StopReason,
		TokensIn:   resp.Usage.InputTokens,
		TokensOut:  resp.Usage.OutputTokens,
		ToolCalls:  []agent.ToolCall{},
	}

	// Extract content and tool calls
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			response.Content += block.Text
		case "tool_use":
			response.ToolCalls = append(response.ToolCalls, agent.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	return response
}

// NewClient creates a new Claude client with default settings
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:      apiKey,
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   4096,
		Temperature: 0.2,
		TopP:        0.3,
		HTTPClient:  &http.Client{Timeout: 60 * time.Second},
	}
}

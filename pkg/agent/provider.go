package agent

import "context"

// Provider is the interface for LLM providers
type Provider interface {
	// Call sends a message to the LLM and returns the response
	Call(ctx context.Context, messages []Message) (*Response, error)
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"` // Message content
}

// Response represents the LLM response
type Response struct {
	Content   string      `json:"content"`    // Response text
	ToolCalls []ToolCall  `json:"tool_calls"` // Tool calls requested
	StopReason string     `json:"stop_reason"` // "end_turn", "tool_use", "max_tokens"
	TokensIn  int         `json:"tokens_in"`  // Input tokens
	TokensOut int         `json:"tokens_out"` // Output tokens
}

// ToolCall represents a tool call request from the LLM
type ToolCall struct {
	ID    string                 `json:"id"`     // Unique ID
	Name  string                 `json:"name"`   // Tool name
	Input map[string]interface{} `json:"input"`  // Tool input parameters
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"` // ID of the tool call
	Content    string `json:"content"`      // Result content
	IsError    bool   `json:"is_error"`     // Whether this is an error
}

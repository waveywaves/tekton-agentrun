package agent

import (
	"context"
	"fmt"
)

// Tool is the interface for agent tools
type Tool interface {
	// Name returns the tool name
	Name() string
	// Execute runs the tool with given input
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
}

// Policy is the interface for policy enforcement
type Policy interface {
	// Allow checks if a tool call is allowed
	Allow(ctx context.Context, toolCall ToolCall) error
}

// Loop implements the plan-act-reflect loop
type Loop struct {
	Provider      Provider
	Tools         map[string]Tool
	Policy        Policy
	Goal          string
	SystemPrompt  string
	MaxIterations int
}

// Result represents the result of running the loop
type Result struct {
	Status        string            `json:"status"` // "succeeded", "failed", "max_iterations"
	Iterations    int               `json:"iterations"`
	ToolCalls     []ToolCallRecord  `json:"tool_calls"`
	FinalResponse string            `json:"final_response"`
	TotalTokensIn int               `json:"total_tokens_in"`
	TotalTokensOut int              `json:"total_tokens_out"`
	Error         string            `json:"error,omitempty"`
}

// ToolCallRecord records a tool call execution
type ToolCallRecord struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Input  map[string]interface{} `json:"input"`
	Output string                 `json:"output"`
	Error  string                 `json:"error,omitempty"`
}

// Run executes the plan-act-reflect loop
func (l *Loop) Run(ctx context.Context) (*Result, error) {
	result := &Result{
		Status:     "succeeded",
		ToolCalls:  []ToolCallRecord{},
	}

	messages := []Message{}

	// Add system prompt if provided
	if l.SystemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: l.SystemPrompt,
		})
	}

	// Add initial goal
	messages = append(messages, Message{
		Role:    "user",
		Content: fmt.Sprintf("Goal: %s\n\nPlease analyze this goal and take the necessary actions to achieve it.", l.Goal),
	})

	for iteration := 0; iteration < l.MaxIterations; iteration++ {
		result.Iterations = iteration + 1

		// Call LLM
		response, err := l.Provider.Call(ctx, messages)
		if err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("LLM call failed: %v", err)
			return result, err
		}

		result.TotalTokensIn += response.TokensIn
		result.TotalTokensOut += response.TokensOut

		// Add assistant response to messages
		messages = append(messages, Message{
			Role:    "assistant",
			Content: response.Content,
		})

		// If no tool calls, continue to next iteration
		if len(response.ToolCalls) == 0 {
			result.FinalResponse = response.Content
			// Add a prompt asking for more analysis or tool usage
			messages = append(messages, Message{
				Role:    "user",
				Content: "Please continue analyzing the goal. If you need more information, use the available tools. If you're confident the goal is achieved, provide your final answer.",
			})
			continue
		}

		// Process tool calls
		toolResults := []ToolResult{}
		for _, toolCall := range response.ToolCalls {
			// Check policy
			if err := l.Policy.Allow(ctx, toolCall); err != nil {
				result.Status = "failed"
				result.Error = fmt.Sprintf("Policy violation for tool %s: %v", toolCall.Name, err)
				return result, fmt.Errorf("policy violation: %w", err)
			}

			// Find tool
			tool, ok := l.Tools[toolCall.Name]
			if !ok {
				result.Status = "failed"
				result.Error = fmt.Sprintf("Tool not found: %s", toolCall.Name)
				return result, fmt.Errorf("tool not found: %s", toolCall.Name)
			}

			// Execute tool
			output, err := tool.Execute(ctx, toolCall.Input)

			record := ToolCallRecord{
				ID:    toolCall.ID,
				Name:  toolCall.Name,
				Input: toolCall.Input,
			}

			if err != nil {
				record.Error = err.Error()
				toolResults = append(toolResults, ToolResult{
					ToolCallID: toolCall.ID,
					Content:    err.Error(),
					IsError:    true,
				})
			} else {
				record.Output = output
				toolResults = append(toolResults, ToolResult{
					ToolCallID: toolCall.ID,
					Content:    output,
					IsError:    false,
				})
			}

			result.ToolCalls = append(result.ToolCalls, record)
		}

		// Add tool results to messages as user message
		// In a real implementation, this would be formatted according to the provider's tool result format
		toolResultsContent := ""
		for _, tr := range toolResults {
			if tr.IsError {
				toolResultsContent += fmt.Sprintf("Tool call %s failed: %s\n", tr.ToolCallID, tr.Content)
			} else {
				toolResultsContent += fmt.Sprintf("Tool call %s result: %s\n", tr.ToolCallID, tr.Content)
			}
		}

		messages = append(messages, Message{
			Role:    "user",
			Content: toolResultsContent,
		})

		// Get reflection from LLM
		reflectResponse, err := l.Provider.Call(ctx, messages)
		if err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("Reflection call failed: %v", err)
			return result, err
		}

		result.TotalTokensIn += reflectResponse.TokensIn
		result.TotalTokensOut += reflectResponse.TokensOut

		// Add reflection to messages
		messages = append(messages, Message{
			Role:    "assistant",
			Content: reflectResponse.Content,
		})

		result.FinalResponse = reflectResponse.Content

		// If reflection has no more tool calls and stop reason is end_turn, we're done
		if len(reflectResponse.ToolCalls) == 0 && reflectResponse.StopReason == "end_turn" {
			result.Status = "succeeded"
			return result, nil
		}
	}

	// Max iterations reached
	result.Status = "max_iterations"
	return result, nil
}

package agent

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"
)

// OPAPolicy implements the Policy interface using Open Policy Agent
type OPAPolicy struct {
	PolicyContent string
	Data          map[string]interface{}
	query         rego.PreparedEvalQuery
}

// Initialize compiles the OPA policy
func (o *OPAPolicy) Initialize() error {
	// Build the query options
	opts := []func(*rego.Rego){
		rego.Query("data.agent.tools.allow"),
		rego.Module("agent.rego", o.PolicyContent),
	}

	// Add data if provided
	if o.Data != nil {
		store := inmem.NewFromObject(o.Data)
		opts = append(opts, rego.Store(store))
	}

	// Build the query
	r := rego.New(opts...)

	// Prepare the query
	query, err := r.PrepareForEval(context.Background())
	if err != nil {
		return fmt.Errorf("failed to compile policy: %w", err)
	}

	o.query = query
	return nil
}

// Allow checks if a tool call is allowed by the policy
func (o *OPAPolicy) Allow(ctx context.Context, toolCall ToolCall) error {
	// Build input for policy evaluation
	input := map[string]interface{}{
		"tool": toolCall.Name,
	}

	// Add all input parameters
	for k, v := range toolCall.Input {
		input[k] = v
	}

	// Evaluate the policy
	results, err := o.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		// Fail closed on evaluation error
		return fmt.Errorf("policy evaluation failed: %w", err)
	}

	// Check if policy allowed the action
	if len(results) == 0 {
		return fmt.Errorf("policy denied: no matching allow rule")
	}

	// Check if the result is true
	allowed, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		return fmt.Errorf("policy denied: invalid result type")
	}

	if !allowed {
		return fmt.Errorf("policy denied: allow rule returned false")
	}

	return nil
}

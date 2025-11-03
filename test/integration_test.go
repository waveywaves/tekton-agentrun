// +build integration

package test

import (
	"context"
	"os"
	"testing"

	"github.com/waveywaves/agentrun-controller/pkg/agent"
	"github.com/waveywaves/agentrun-controller/pkg/providers/claude"
	"github.com/waveywaves/agentrun-controller/pkg/tools/tekton"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAgentCreatesPipelineRun(t *testing.T) {
	// Get Claude API key from environment
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("CLAUDE_API_KEY not set")
	}

	// Create fake Kubernetes and Tekton clients
	kubeClient := fake.NewSimpleClientset()
	tektonClient := tektonfake.NewSimpleClientset()

	// Create Claude provider with tool definitions
	provider := claude.NewClient(apiKey)
	provider.Tools = []claude.Tool{
		{
			Name:        "tekton_create_pipelinerun",
			Description: "Create a Tekton PipelineRun to execute a Pipeline with specific parameters",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes namespace where the PipelineRun will be created",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name for the PipelineRun",
					},
					"pipelineName": map[string]interface{}{
						"type":        "string",
						"description": "Name of the existing Pipeline to run",
					},
					"params": map[string]interface{}{
						"type":        "array",
						"description": "Array of parameter objects with name and value fields",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name":  map[string]interface{}{"type": "string"},
								"value": map[string]interface{}{"type": "string"},
							},
							"required": []string{"name", "value"},
						},
					},
				},
				"required": []string{"namespace", "name", "pipelineName"},
			},
		},
	}

	// Create Tekton tool
	tektonTool := &tekton.CreatePipelineRun{
		KubeClient:   kubeClient,
		TektonClient: tektonClient,
	}

	// Create OPA policy that allows PipelineRun creation
	policy := &agent.OPAPolicy{
		PolicyContent: `
package agent.tools
default allow = false
allow {
    input.tool == "tekton_create_pipelinerun"
    input.namespace == "default"
}
`,
	}
	err := policy.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize policy: %v", err)
	}

	// Create agent loop
	loop := &agent.Loop{
		Provider: provider,
		Tools: map[string]agent.Tool{
			"tekton_create_pipelinerun": tektonTool,
		},
		Policy: policy,
		SystemPrompt: `You are a Kubernetes agent. Use the available tools to complete the user's goal.`,
		Goal: `Create a PipelineRun in the default namespace named "test-build-run-1" that runs the Pipeline "buildpacks" with a parameter "image" set to "myapp:v1.0.0".`,
		MaxIterations: 3,
	}

	// Run the agent
	ctx := context.Background()
	result, err := loop.Run(ctx)
	if err != nil {
		t.Fatalf("Agent failed: %v", err)
	}

	t.Logf("Agent completed with status: %s", result.Status)
	t.Logf("Iterations: %d", result.Iterations)
	t.Logf("Tool calls: %d", len(result.ToolCalls))
	t.Logf("Tokens: in=%d, out=%d", result.TotalTokensIn, result.TotalTokensOut)

	// Verify a tool call was made
	if len(result.ToolCalls) == 0 {
		t.Fatal("Expected at least one tool call")
	}

	// Verify it was a tekton_create_pipelinerun call
	foundPipelineRunCreation := false
	for _, tc := range result.ToolCalls {
		t.Logf("Tool call: %s", tc.Name)
		if tc.Name == "tekton_create_pipelinerun" {
			foundPipelineRunCreation = true
			t.Logf("PipelineRun creation input: %+v", tc.Input)
			t.Logf("PipelineRun creation output: %s", tc.Output)
		}
	}

	if !foundPipelineRunCreation {
		t.Error("Expected tekton_create_pipelinerun tool to be called")
	}

	// Verify PipelineRun was actually created in fake client
	prs, err := tektonClient.TektonV1().PipelineRuns("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list PipelineRuns: %v", err)
	}

	if len(prs.Items) == 0 {
		t.Error("Expected at least one PipelineRun to be created")
	} else {
		for _, pr := range prs.Items {
			t.Logf("Created PipelineRun: %s (Pipeline: %s)", pr.Name, pr.Spec.PipelineRef.Name)
			if len(pr.Spec.Params) > 0 {
				t.Logf("Parameters: %+v", pr.Spec.Params)
			}
		}
	}
}

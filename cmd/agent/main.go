package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/waveywaves/agentrun-controller/pkg/agent"
	"github.com/waveywaves/agentrun-controller/pkg/providers/claude"
	"github.com/waveywaves/agentrun-controller/pkg/tools/k8s"
	"github.com/waveywaves/agentrun-controller/pkg/tools/tekton"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	goal          string
	maxIterations int
	timeout       time.Duration
	provider      string
	configPath    string
	dataPath      string
	secretsPath   string
)

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func main() {
	flag.StringVar(&goal, "goal", os.Getenv("AGENTRUN_GOAL"), "Goal for the agent to achieve")
	flag.IntVar(&maxIterations, "max-iterations", 3, "Maximum iterations for plan-act-reflect loop")
	flag.DurationVar(&timeout, "timeout", 8*time.Minute, "Timeout for agent execution")
	flag.StringVar(&provider, "provider", getEnvOrDefault("LLM_PROVIDER", "claude"), "LLM provider (claude or gemini)")
	flag.StringVar(&configPath, "config-path", "/workspace/config", "Path to config volume")
	flag.StringVar(&dataPath, "data-path", "/workspace/data", "Path to data volume")
	flag.StringVar(&secretsPath, "secrets-path", "/workspace/secrets", "Path to secrets volume")
	flag.Parse()

	if goal == "" {
		log.Fatal("Goal is required (--goal or AGENTRUN_GOAL env var)")
	}

	log.Printf("Agent starting with goal: %s", goal)
	log.Printf("Max iterations: %d, Timeout: %v, Provider: %s", maxIterations, timeout, provider)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Load system prompt
	systemPrompt, err := loadSystemPrompt(configPath)
	if err != nil {
		log.Fatalf("Failed to load system prompt: %v", err)
	}
	log.Println("System prompt loaded")

	// Load OPA policy
	policyContent, err := loadOPAPolicy(configPath)
	if err != nil {
		log.Fatalf("Failed to load OPA policy: %v", err)
	}
	log.Println("OPA policy loaded")

	// Initialize OPA policy
	policy := &agent.OPAPolicy{
		PolicyContent: policyContent,
	}
	if err := policy.Initialize(); err != nil {
		log.Fatalf("Failed to initialize OPA policy: %v", err)
	}
	log.Println("OPA policy initialized")

	// Build Kubernetes clients
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get in-cluster config: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	tektonClient, err := tektonclient.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Tekton client: %v", err)
	}

	// Set up tools
	tools := map[string]agent.Tool{
		"k8s_get_resources": &k8s.GetResources{
			KubeClient: kubeClient,
		},
		"k8s_get_logs": &k8s.GetLogs{
			KubeClient: kubeClient,
		},
		"tekton_create_pipelinerun": &tekton.CreatePipelineRun{
			KubeClient:   kubeClient,
			TektonClient: tektonClient,
			AgentRunName: os.Getenv("AGENTRUN_NAME"),
			AgentRunUID:  types.UID(os.Getenv("AGENTRUN_UID")),
		},
	}
	log.Printf("Tools registered: %d", len(tools))

	// Set up LLM provider
	var llmProvider agent.Provider
	switch provider {
	case "claude":
		apiKey, err := loadSecret(secretsPath, "CLAUDE_API_KEY")
		if err != nil {
			log.Fatalf("Failed to load Claude API key: %v", err)
		}
		claudeClient := claude.NewClient(apiKey)
		claudeClient.Tools = buildClaudeTools()
		llmProvider = claudeClient
		log.Println("Claude provider initialized")
	default:
		log.Fatalf("Unsupported provider: %s", provider)
	}

	// Create agent loop
	loop := &agent.Loop{
		Provider:      llmProvider,
		Tools:         tools,
		Policy:        policy,
		SystemPrompt:  systemPrompt,
		Goal:          goal,
		MaxIterations: maxIterations,
	}

	// Run agent
	log.Println("Starting agent execution...")
	result, err := loop.Run(ctx)
	if err != nil {
		log.Printf("Agent execution failed: %v", err)
		saveResult(dataPath, result, err)
		os.Exit(1)
	}

	log.Printf("Agent execution completed: status=%s, iterations=%d", result.Status, result.Iterations)
	log.Printf("Tool calls: %d, Tokens: in=%d out=%d", len(result.ToolCalls), result.TotalTokensIn, result.TotalTokensOut)

	// Save result
	if err := saveResult(dataPath, result, nil); err != nil {
		log.Printf("Warning: Failed to save result: %v", err)
	}

	if result.Status != "succeeded" {
		os.Exit(1)
	}
}

func loadSystemPrompt(configPath string) (string, error) {
	path := filepath.Join(configPath, "prompts", "system.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read system prompt: %w", err)
	}
	return string(data), nil
}

func loadOPAPolicy(configPath string) (string, error) {
	// Try to load policy from guardrails directory
	path := filepath.Join(configPath, "guardrails", "policy.rego")
	data, err := os.ReadFile(path)
	if err != nil {
		// If not found, use a default permissive policy
		log.Printf("No policy found at %s, using default policy", path)
		return `
package agent.tools
default allow = true
`, nil
	}
	return string(data), nil
}

func loadSecret(secretsPath, key string) (string, error) {
	path := filepath.Join(secretsPath, key)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret %s: %w", key, err)
	}
	return string(data), nil
}

func buildClaudeTools() []claude.Tool {
	return []claude.Tool{
		{
			Name:        "k8s_get_resources",
			Description: "List Kubernetes resources like pods, deployments, services, or replicasets in a namespace",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes namespace to query",
					},
					"resourceType": map[string]interface{}{
						"type":        "string",
						"description": "Type of resource: pods, deployments, services, or replicasets",
					},
					"labelSelector": map[string]interface{}{
						"type":        "string",
						"description": "Optional label selector to filter resources",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of resources to return (default 100)",
					},
				},
				"required": []string{"namespace", "resourceType"},
			},
		},
		{
			Name:        "k8s_get_logs",
			Description: "Fetch logs from a Kubernetes pod",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes namespace",
					},
					"pod": map[string]interface{}{
						"type":        "string",
						"description": "Pod name",
					},
					"container": map[string]interface{}{
						"type":        "string",
						"description": "Container name (optional, required for multi-container pods)",
					},
					"tailLines": map[string]interface{}{
						"type":        "number",
						"description": "Number of lines to tail (max 500)",
					},
					"sinceSeconds": map[string]interface{}{
						"type":        "number",
						"description": "Return logs newer than this duration in seconds (max 900)",
					},
				},
				"required": []string{"namespace", "pod"},
			},
		},
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
						"description": "Name for the PipelineRun (should be unique and descriptive)",
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
					"workspaces": map[string]interface{}{
						"type":        "array",
						"description": "Array of workspace bindings",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name":     map[string]interface{}{"type": "string"},
								"pvcName":  map[string]interface{}{"type": "string"},
								"emptyDir": map[string]interface{}{"type": "boolean"},
							},
							"required": []string{"name"},
						},
					},
				},
				"required": []string{"namespace", "name", "pipelineName"},
			},
		},
	}
}

func saveResult(dataPath string, result *agent.Result, execError error) error {
	output := map[string]interface{}{
		"status":      result.Status,
		"iterations":  result.Iterations,
		"toolCalls":   result.ToolCalls,
		"tokensIn":    result.TotalTokensIn,
		"tokensOut":   result.TotalTokensOut,
		"response":    result.FinalResponse,
	}
	if execError != nil {
		output["error"] = execError.Error()
	}
	if result.Error != "" {
		output["agentError"] = result.Error
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	resultPath := filepath.Join(dataPath, "result.json")
	if err := os.WriteFile(resultPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	log.Printf("Result saved to %s", resultPath)
	return nil
}

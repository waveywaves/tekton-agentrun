package tekton

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
)

func TestCreatePipelineRun_Basic(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pipelinerun with required fields",
			input: map[string]interface{}{
				"namespace":    "default",
				"name":         "test-run",
				"pipelineName": "test-pipeline",
			},
			wantErr: false,
		},
		{
			name: "missing namespace",
			input: map[string]interface{}{
				"name":         "test-run",
				"pipelineName": "test-pipeline",
			},
			wantErr: true,
			errMsg:  "namespace is required",
		},
		{
			name: "missing name",
			input: map[string]interface{}{
				"namespace":    "default",
				"pipelineName": "test-pipeline",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing pipelineName",
			input: map[string]interface{}{
				"namespace": "default",
				"name":      "test-run",
			},
			wantErr: true,
			errMsg:  "pipelineName is required",
		},
		{
			name: "with params",
			input: map[string]interface{}{
				"namespace":    "default",
				"name":         "test-run",
				"pipelineName": "test-pipeline",
				"params": []interface{}{
					map[string]interface{}{
						"name":  "image",
						"value": "nginx:latest",
					},
					map[string]interface{}{
						"name":  "replicas",
						"value": "3",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with workspace",
			input: map[string]interface{}{
				"namespace":    "default",
				"name":         "test-run",
				"pipelineName": "test-pipeline",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name":    "source",
						"pvcName": "source-pvc",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh clients for each test case
			kubeClient := fake.NewSimpleClientset()
			tektonClient := tektonfake.NewSimpleClientset()

			tool := &CreatePipelineRun{
				KubeClient:   kubeClient,
				TektonClient: tektonClient,
			}

			ctx := context.Background()
			output, err := tool.Execute(ctx, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message %q does not contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			// Verify output contains success message
			if !strings.Contains(output, "created successfully") {
				t.Errorf("Output does not contain success message. Output: %s", output)
			}

			// Verify PipelineRun was created
			prName := tt.input["name"].(string)
			prNamespace := tt.input["namespace"].(string)
			pr, err := tektonClient.TektonV1().PipelineRuns(prNamespace).Get(ctx, prName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Failed to get created PipelineRun: %v", err)
			}

			// Verify pipeline reference
			if pr.Spec.PipelineRef.Name != tt.input["pipelineName"].(string) {
				t.Errorf("PipelineRef.Name = %v, want %v", pr.Spec.PipelineRef.Name, tt.input["pipelineName"])
			}

			// Verify params if provided
			if params, ok := tt.input["params"]; ok {
				paramsList := params.([]interface{})
				if len(pr.Spec.Params) != len(paramsList) {
					t.Errorf("Params length = %d, want %d", len(pr.Spec.Params), len(paramsList))
				}
			}

			// Verify workspaces if provided
			if workspaces, ok := tt.input["workspaces"]; ok {
				workspacesList := workspaces.([]interface{})
				if len(pr.Spec.Workspaces) != len(workspacesList) {
					t.Errorf("Workspaces length = %d, want %d", len(pr.Spec.Workspaces), len(workspacesList))
				}
			}
		})
	}
}

func TestCreatePipelineRun_Name(t *testing.T) {
	tool := &CreatePipelineRun{}
	if got := tool.Name(); got != "tekton_create_pipelinerun" {
		t.Errorf("Name() = %v, want tekton_create_pipelinerun", got)
	}
}

func TestCreatePipelineRun_OwnerReference(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	tektonClient := tektonfake.NewSimpleClientset()

	tool := &CreatePipelineRun{
		KubeClient:   kubeClient,
		TektonClient: tektonClient,
		AgentRunName: "test-agentrun",
		AgentRunUID:  "test-uid",
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"namespace":    "default",
		"name":         "test-run",
		"pipelineName": "test-pipeline",
	}

	_, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify PipelineRun has owner reference
	pr, err := tektonClient.TektonV1().PipelineRuns("default").Get(ctx, "test-run", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created PipelineRun: %v", err)
	}

	if len(pr.OwnerReferences) == 0 {
		t.Error("PipelineRun should have owner reference")
	}
}

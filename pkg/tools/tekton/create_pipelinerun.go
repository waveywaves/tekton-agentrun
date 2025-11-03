package tekton

import (
	"context"
	"fmt"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// CreatePipelineRun implements the tekton_create_pipelinerun tool
type CreatePipelineRun struct {
	KubeClient   kubernetes.Interface
	TektonClient tektonclient.Interface
	AgentRunName string
	AgentRunUID  types.UID
}

// Name returns the tool name
func (c *CreatePipelineRun) Name() string {
	return "tekton_create_pipelinerun"
}

// Execute runs the tool
func (c *CreatePipelineRun) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	// Parse input
	namespace, ok := input["namespace"].(string)
	if !ok || namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	name, ok := input["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required")
	}

	pipelineName, ok := input["pipelineName"].(string)
	if !ok || pipelineName == "" {
		return "", fmt.Errorf("pipelineName is required")
	}

	// Build PipelineRun
	pr := &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: tektonv1.PipelineRunSpec{
			PipelineRef: &tektonv1.PipelineRef{
				Name: pipelineName,
			},
		},
	}

	// Add owner reference if AgentRun info is provided
	if c.AgentRunName != "" && c.AgentRunUID != "" {
		pr.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "agent.tekton.dev/v1alpha1",
				Kind:       "AgentRun",
				Name:       c.AgentRunName,
				UID:        c.AgentRunUID,
			},
		}
	}

	// Parse params if provided
	if params, ok := input["params"]; ok {
		paramsList, ok := params.([]interface{})
		if !ok {
			return "", fmt.Errorf("params must be an array")
		}

		for _, p := range paramsList {
			paramMap, ok := p.(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("each param must be an object with name and value")
			}

			paramName, ok := paramMap["name"].(string)
			if !ok {
				return "", fmt.Errorf("param name is required")
			}

			paramValue, ok := paramMap["value"].(string)
			if !ok {
				return "", fmt.Errorf("param value must be a string")
			}

			pr.Spec.Params = append(pr.Spec.Params, tektonv1.Param{
				Name: paramName,
				Value: tektonv1.ParamValue{
					Type:      tektonv1.ParamTypeString,
					StringVal: paramValue,
				},
			})
		}
	}

	// Parse workspaces if provided
	if workspaces, ok := input["workspaces"]; ok {
		workspacesList, ok := workspaces.([]interface{})
		if !ok {
			return "", fmt.Errorf("workspaces must be an array")
		}

		for _, w := range workspacesList {
			workspaceMap, ok := w.(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("each workspace must be an object")
			}

			workspaceName, ok := workspaceMap["name"].(string)
			if !ok {
				return "", fmt.Errorf("workspace name is required")
			}

			workspace := tektonv1.WorkspaceBinding{
				Name: workspaceName,
			}

			// Check for PVC
			if pvcName, ok := workspaceMap["pvcName"].(string); ok {
				workspace.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				}
			}

			// Check for EmptyDir
			if emptyDir, ok := workspaceMap["emptyDir"].(bool); ok && emptyDir {
				workspace.EmptyDir = &corev1.EmptyDirVolumeSource{}
			}

			pr.Spec.Workspaces = append(pr.Spec.Workspaces, workspace)
		}
	}

	// Create PipelineRun
	created, err := c.TektonClient.TektonV1().PipelineRuns(namespace).Create(ctx, pr, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create PipelineRun: %w", err)
	}

	return fmt.Sprintf("PipelineRun %s/%s created successfully", created.Namespace, created.Name), nil
}

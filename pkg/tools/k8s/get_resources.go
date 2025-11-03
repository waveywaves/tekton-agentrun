package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetResources implements the k8s_get_resources tool
type GetResources struct {
	KubeClient kubernetes.Interface
}

// Name returns the tool name
func (g *GetResources) Name() string {
	return "k8s_get_resources"
}

// Execute runs the tool
func (g *GetResources) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	// Parse input
	namespace, ok := input["namespace"].(string)
	if !ok || namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	resourceType, ok := input["resourceType"].(string)
	if !ok || resourceType == "" {
		return "", fmt.Errorf("resourceType is required")
	}

	labelSelector := ""
	if ls, ok := input["labelSelector"].(string); ok {
		labelSelector = ls
	}

	limit := int64(100) // Default limit
	if l, ok := input["limit"].(float64); ok {
		limit = int64(l)
	} else if l, ok := input["limit"].(int); ok {
		limit = int64(l)
	}

	if limit > 100 {
		limit = 100
	}

	// Get resources based on type
	switch resourceType {
	case "pods":
		return g.getPods(ctx, namespace, labelSelector, limit)
	case "deployments":
		return g.getDeployments(ctx, namespace, labelSelector, limit)
	case "services":
		return g.getServices(ctx, namespace, labelSelector, limit)
	case "replicasets":
		return g.getReplicaSets(ctx, namespace, labelSelector, limit)
	default:
		return "", fmt.Errorf("invalid resourceType: %s (must be one of: pods, deployments, services, replicasets)", resourceType)
	}
}

func (g *GetResources) getPods(ctx context.Context, namespace, labelSelector string, limit int64) (string, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: labelSelector,
		Limit:         limit,
	}

	pods, err := g.KubeClient.CoreV1().Pods(namespace).List(ctx, listOpts)
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	type podInfo struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Phase     string            `json:"phase"`
		Labels    map[string]string `json:"labels,omitempty"`
		Ready     string            `json:"ready"`
		Restarts  int32             `json:"restarts"`
		Age       string            `json:"age"`
	}

	// Truncate results to limit (fake client doesn't respect Limit)
	items := pods.Items
	if int64(len(items)) > limit {
		items = items[:limit]
	}

	result := make([]podInfo, 0, len(items))
	for _, pod := range items {
		// Count ready containers
		readyCount := 0
		totalCount := len(pod.Status.ContainerStatuses)
		restarts := int32(0)

		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
			restarts += cs.RestartCount
		}

		age := "unknown"
		if !pod.CreationTimestamp.IsZero() {
			age = pod.CreationTimestamp.String()
		}

		result = append(result, podInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
			Labels:    pod.Labels,
			Ready:     fmt.Sprintf("%d/%d", readyCount, totalCount),
			Restarts:  restarts,
			Age:       age,
		})
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(output), nil
}

func (g *GetResources) getDeployments(ctx context.Context, namespace, labelSelector string, limit int64) (string, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: labelSelector,
		Limit:         limit,
	}

	deployments, err := g.KubeClient.AppsV1().Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		return "", fmt.Errorf("failed to list deployments: %w", err)
	}

	type deploymentInfo struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Replicas  string            `json:"replicas"`
		Labels    map[string]string `json:"labels,omitempty"`
		Age       string            `json:"age"`
	}

	// Truncate results to limit (fake client doesn't respect Limit)
	items := deployments.Items
	if int64(len(items)) > limit {
		items = items[:limit]
	}

	result := make([]deploymentInfo, 0, len(items))
	for _, deploy := range items {
		replicas := "0/0"
		if deploy.Spec.Replicas != nil {
			replicas = fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
		}

		age := "unknown"
		if !deploy.CreationTimestamp.IsZero() {
			age = deploy.CreationTimestamp.String()
		}

		result = append(result, deploymentInfo{
			Name:      deploy.Name,
			Namespace: deploy.Namespace,
			Replicas:  replicas,
			Labels:    deploy.Labels,
			Age:       age,
		})
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(output), nil
}

func (g *GetResources) getServices(ctx context.Context, namespace, labelSelector string, limit int64) (string, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: labelSelector,
		Limit:         limit,
	}

	services, err := g.KubeClient.CoreV1().Services(namespace).List(ctx, listOpts)
	if err != nil {
		return "", fmt.Errorf("failed to list services: %w", err)
	}

	type serviceInfo struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Type      string            `json:"type"`
		ClusterIP string            `json:"cluster_ip"`
		Ports     []string          `json:"ports"`
		Labels    map[string]string `json:"labels,omitempty"`
		Age       string            `json:"age"`
	}

	// Truncate results to limit (fake client doesn't respect Limit)
	items := services.Items
	if int64(len(items)) > limit {
		items = items[:limit]
	}

	result := make([]serviceInfo, 0, len(items))
	for _, svc := range items {
		ports := make([]string, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}

		age := "unknown"
		if !svc.CreationTimestamp.IsZero() {
			age = svc.CreationTimestamp.String()
		}

		result = append(result, serviceInfo{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Type:      string(svc.Spec.Type),
			ClusterIP: svc.Spec.ClusterIP,
			Ports:     ports,
			Labels:    svc.Labels,
			Age:       age,
		})
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(output), nil
}

func (g *GetResources) getReplicaSets(ctx context.Context, namespace, labelSelector string, limit int64) (string, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: labelSelector,
		Limit:         limit,
	}

	replicaSets, err := g.KubeClient.AppsV1().ReplicaSets(namespace).List(ctx, listOpts)
	if err != nil {
		return "", fmt.Errorf("failed to list replicasets: %w", err)
	}

	type replicaSetInfo struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Replicas  string            `json:"replicas"`
		Labels    map[string]string `json:"labels,omitempty"`
		Age       string            `json:"age"`
	}

	// Truncate results to limit (fake client doesn't respect Limit)
	items := replicaSets.Items
	if int64(len(items)) > limit {
		items = items[:limit]
	}

	result := make([]replicaSetInfo, 0, len(items))
	for _, rs := range items {
		replicas := "0/0"
		if rs.Spec.Replicas != nil {
			replicas = fmt.Sprintf("%d/%d", rs.Status.ReadyReplicas, *rs.Spec.Replicas)
		}

		age := "unknown"
		if !rs.CreationTimestamp.IsZero() {
			age = rs.CreationTimestamp.String()
		}

		result = append(result, replicaSetInfo{
			Name:      rs.Name,
			Namespace: rs.Namespace,
			Replicas:  replicas,
			Labels:    rs.Labels,
			Age:       age,
		})
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(output), nil
}

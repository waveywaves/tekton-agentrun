package k8s

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// GetLogs implements the k8s_get_logs tool
type GetLogs struct {
	KubeClient kubernetes.Interface
}

// Name returns the tool name
func (g *GetLogs) Name() string {
	return "k8s_get_logs"
}

// Execute runs the tool
func (g *GetLogs) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	// Parse input
	namespace, ok := input["namespace"].(string)
	if !ok || namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	pod, ok := input["pod"].(string)
	if !ok || pod == "" {
		return "", fmt.Errorf("pod is required")
	}

	container := ""
	if c, ok := input["container"].(string); ok {
		container = c
	}

	tailLines := int64(500) // Default
	if tl, ok := input["tailLines"].(float64); ok {
		tailLines = int64(tl)
	} else if tl, ok := input["tailLines"].(int); ok {
		tailLines = int64(tl)
	} else if tl, ok := input["tailLines"].(int64); ok {
		tailLines = tl
	}

	if tailLines > 500 {
		tailLines = 500
	}

	sinceSeconds := int64(900) // Default: 15 minutes
	if ss, ok := input["sinceSeconds"].(float64); ok {
		sinceSeconds = int64(ss)
	} else if ss, ok := input["sinceSeconds"].(int); ok {
		sinceSeconds = int64(ss)
	} else if ss, ok := input["sinceSeconds"].(int64); ok {
		sinceSeconds = ss
	}

	if sinceSeconds > 900 {
		sinceSeconds = 900
	}

	// Build log options
	logOpts := &corev1.PodLogOptions{
		Container:    container,
		TailLines:    &tailLines,
		SinceSeconds: &sinceSeconds,
	}

	// Get logs
	req := g.KubeClient.CoreV1().Pods(namespace).GetLogs(pod, logOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer podLogs.Close()

	// Read logs
	buf, err := io.ReadAll(podLogs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return string(buf), nil
}

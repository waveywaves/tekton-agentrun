package k8s

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetLogs_Basic(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	kubeClient := fake.NewSimpleClientset(pod)

	tool := &GetLogs{
		KubeClient: kubeClient,
	}

	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid input with pod and namespace",
			input: map[string]interface{}{
				"namespace": "default",
				"pod":       "test-pod",
			},
			wantErr: false,
		},
		{
			name: "valid input with container specified",
			input: map[string]interface{}{
				"namespace": "default",
				"pod":       "test-pod",
				"container": "main",
			},
			wantErr: false,
		},
		{
			name: "valid input with tailLines",
			input: map[string]interface{}{
				"namespace": "default",
				"pod":       "test-pod",
				"tailLines": float64(100),
			},
			wantErr: false,
		},
		{
			name: "valid input with sinceSeconds",
			input: map[string]interface{}{
				"namespace":    "default",
				"pod":          "test-pod",
				"sinceSeconds": float64(600),
			},
			wantErr: false,
		},
		{
			name: "missing namespace",
			input: map[string]interface{}{
				"pod": "test-pod",
			},
			wantErr: true,
		},
		{
			name: "missing pod",
			input: map[string]interface{}{
				"namespace": "default",
			},
			wantErr: true,
		},
		{
			name: "tailLines exceeds max",
			input: map[string]interface{}{
				"namespace": "default",
				"pod":       "test-pod",
				"tailLines": float64(1000),
			},
			wantErr: false, // Should cap to max, not error
		},
		{
			name: "sinceSeconds exceeds max",
			input: map[string]interface{}{
				"namespace":    "default",
				"pod":          "test-pod",
				"sinceSeconds": float64(2000),
			},
			wantErr: false, // Should cap to max, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := tool.Execute(ctx, tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetLogs_Name(t *testing.T) {
	tool := &GetLogs{}
	if got := tool.Name(); got != "k8s_get_logs" {
		t.Errorf("Name() = %v, want k8s_get_logs", got)
	}
}

func TestGetLogs_Validation(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()

	tool := &GetLogs{
		KubeClient: kubeClient,
	}

	tests := []struct {
		name        string
		input       map[string]interface{}
		wantErr     bool
		errContains string
	}{
		{
			name: "empty namespace",
			input: map[string]interface{}{
				"namespace": "",
				"pod":       "test-pod",
			},
			wantErr:     true,
			errContains: "namespace is required",
		},
		{
			name: "empty pod",
			input: map[string]interface{}{
				"namespace": "default",
				"pod":       "",
			},
			wantErr:     true,
			errContains: "pod is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := tool.Execute(ctx, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Error message %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

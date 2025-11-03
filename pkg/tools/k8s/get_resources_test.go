package k8s

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetResources_Pods(t *testing.T) {
	// Create fake client with test pods
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "default",
			Labels: map[string]string{
				"app": "other",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	kubeClient := fake.NewSimpleClientset(pod1, pod2)

	tool := &GetResources{
		KubeClient: kubeClient,
	}

	tests := []struct {
		name          string
		input         map[string]interface{}
		wantErr       bool
		wantContains  []string
	}{
		{
			name: "get all pods",
			input: map[string]interface{}{
				"namespace":    "default",
				"resourceType": "pods",
			},
			wantErr:      false,
			wantContains: []string{"pod1", "pod2", "Running", "Pending"},
		},
		{
			name: "get pods with label selector",
			input: map[string]interface{}{
				"namespace":     "default",
				"resourceType":  "pods",
				"labelSelector": "app=test",
			},
			wantErr:      false,
			wantContains: []string{"pod1", "Running"},
		},
		{
			name: "missing namespace",
			input: map[string]interface{}{
				"resourceType": "pods",
			},
			wantErr: true,
		},
		{
			name: "missing resourceType",
			input: map[string]interface{}{
				"namespace": "default",
			},
			wantErr: true,
		},
		{
			name: "invalid resourceType",
			input: map[string]interface{}{
				"namespace":    "default",
				"resourceType": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			output, err := tool.Execute(ctx, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Output does not contain %q. Output: %s", want, output)
				}
			}
		})
	}
}

func TestGetResources_Deployments(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     2,
			AvailableReplicas: 2,
		},
	}

	kubeClient := fake.NewSimpleClientset(deployment)

	tool := &GetResources{
		KubeClient: kubeClient,
	}

	ctx := context.Background()
	output, err := tool.Execute(ctx, map[string]interface{}{
		"namespace":    "default",
		"resourceType": "deployments",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(output, "test-deployment") {
		t.Errorf("Output does not contain deployment name. Output: %s", output)
	}

	if !strings.Contains(output, "2/3") {
		t.Errorf("Output does not contain replica count. Output: %s", output)
	}
}

func TestGetResources_Services(t *testing.T) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	kubeClient := fake.NewSimpleClientset(service)

	tool := &GetResources{
		KubeClient: kubeClient,
	}

	ctx := context.Background()
	output, err := tool.Execute(ctx, map[string]interface{}{
		"namespace":    "default",
		"resourceType": "services",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(output, "test-service") {
		t.Errorf("Output does not contain service name. Output: %s", output)
	}

	if !strings.Contains(output, "ClusterIP") {
		t.Errorf("Output does not contain service type. Output: %s", output)
	}
}

func TestGetResources_Limit(t *testing.T) {
	kubeClient := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-b", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-c", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-d", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-e", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-f", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-g", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-h", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-i", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-j", Namespace: "default"}},
	)

	tool := &GetResources{
		KubeClient: kubeClient,
	}

	ctx := context.Background()
	output, err := tool.Execute(ctx, map[string]interface{}{
		"namespace":    "default",
		"resourceType": "pods",
		"limit":        5,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should only show 5 pods
	count := 0
	for i := 0; i < 10; i++ {
		if strings.Contains(output, "pod-"+string(rune('a'+i))) {
			count++
		}
	}

	if count > 5 {
		t.Errorf("Output contains %d pods, want max 5. Output: %s", count, output)
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}

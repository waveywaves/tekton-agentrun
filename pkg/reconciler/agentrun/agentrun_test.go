package agentrun

import (
	"context"
	"testing"

	"github.com/waveywaves/agentrun-controller/pkg/apis/agent/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestReconcile_NewAgentRun(t *testing.T) {
	tests := []struct {
		name         string
		agentRun     *v1alpha1.AgentRun
		agentConfig  *v1alpha1.AgentConfig
		wantPhase    string
		wantPodCount int
	}{
		{
			name: "new agentrun creates pod",
			agentRun: &v1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
					UID:       "test-uid",
				},
				Spec: v1alpha1.AgentRunSpec{
					ConfigRef: v1alpha1.ConfigRef{Name: "test-config"},
					Goal:      "Test goal",
				},
				Status: v1alpha1.AgentRunStatus{
					Phase: v1alpha1.AgentRunPhasePending,
				},
			},
			agentConfig: &v1alpha1.AgentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-config",
				},
				Spec: v1alpha1.AgentConfigSpec{
					ServiceAccount: "default",
					ConfigPVC:      "test-config-pvc",
					Provider:       "claude",
					MaxIterations:  3,
				},
			},
			wantPhase:    v1alpha1.AgentRunPhaseActing,
			wantPodCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clients
			kubeClient := fake.NewSimpleClientset()

			// Create reconciler
			r := &Reconciler{
				KubeClient: kubeClient,
				Image:      "agentrun-runtime:test",
			}

			// Store the agentConfig in a map (simulating a cache)
			r.agentConfigs = map[string]*v1alpha1.AgentConfig{
				tt.agentConfig.Name: tt.agentConfig,
			}

			// Reconcile
			ctx := context.Background()
			err := r.Reconcile(ctx, tt.agentRun)

			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			// Check phase
			if tt.agentRun.Status.Phase != tt.wantPhase {
				t.Errorf("Phase = %v, want %v", tt.agentRun.Status.Phase, tt.wantPhase)
			}

			// Check if pod was created
			pods, err := kubeClient.CoreV1().Pods(tt.agentRun.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Failed to list pods: %v", err)
			}

			if len(pods.Items) != tt.wantPodCount {
				t.Errorf("Pod count = %d, want %d", len(pods.Items), tt.wantPodCount)
			}
		})
	}
}

func TestReconcile_AlreadyDone(t *testing.T) {
	agentRun := &v1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-run",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1alpha1.AgentRunSpec{
			ConfigRef: v1alpha1.ConfigRef{Name: "test-config"},
			Goal:      "Test goal",
		},
		Status: v1alpha1.AgentRunStatus{
			Phase: v1alpha1.AgentRunPhaseSucceeded,
		},
	}

	kubeClient := fake.NewSimpleClientset()

	r := &Reconciler{
		KubeClient: kubeClient,
		Image:      "agentrun-runtime:test",
		agentConfigs: map[string]*v1alpha1.AgentConfig{
			"test-config": {
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-config",
				},
				Spec: v1alpha1.AgentConfigSpec{
					ServiceAccount: "default",
					ConfigPVC:      "test-config-pvc",
					Provider:       "claude",
				},
			},
		},
	}

	ctx := context.Background()
	err := r.Reconcile(ctx, agentRun)

	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	// Should not create any pods
	pods, err := kubeClient.CoreV1().Pods(agentRun.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list pods: %v", err)
	}

	if len(pods.Items) != 0 {
		t.Errorf("Pod count = %d, want 0 (should not create pods for completed runs)", len(pods.Items))
	}

	// Phase should remain unchanged
	if agentRun.Status.Phase != v1alpha1.AgentRunPhaseSucceeded {
		t.Errorf("Phase changed to %v, want Succeeded", agentRun.Status.Phase)
	}
}

func TestReconcile_UpdateStatusFromPod(t *testing.T) {
	agentRun := &v1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-run",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1alpha1.AgentRunSpec{
			ConfigRef: v1alpha1.ConfigRef{Name: "test-config"},
			Goal:      "Test goal",
		},
		Status: v1alpha1.AgentRunStatus{
			Phase: v1alpha1.AgentRunPhaseActing,
		},
	}

	// Create a completed pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-run-agent",
			Namespace: "default",
			Labels: map[string]string{
				"agent.tekton.dev/agentrun": "test-run",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
		},
	}

	kubeClient := fake.NewSimpleClientset(pod)

	r := &Reconciler{
		KubeClient: kubeClient,
		Image:      "agentrun-runtime:test",
		agentConfigs: map[string]*v1alpha1.AgentConfig{
			"test-config": {
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-config",
				},
				Spec: v1alpha1.AgentConfigSpec{
					ServiceAccount: "default",
					ConfigPVC:      "test-config-pvc",
					Provider:       "claude",
				},
			},
		},
	}

	ctx := context.Background()
	err := r.Reconcile(ctx, agentRun)

	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	// Should update phase to Succeeded
	if agentRun.Status.Phase != v1alpha1.AgentRunPhaseSucceeded {
		t.Errorf("Phase = %v, want Succeeded", agentRun.Status.Phase)
	}

	// Should set completion time
	if agentRun.Status.CompletionTime == nil {
		t.Error("CompletionTime should be set")
	}
}

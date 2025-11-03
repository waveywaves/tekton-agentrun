package pod

import (
	"testing"

	"github.com/waveywaves/agentrun-controller/pkg/apis/agent/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		name        string
		agentRun    *v1alpha1.AgentRun
		agentConfig *v1alpha1.AgentConfig
		image       string
		checkPod    func(*corev1.Pod) error
	}{
		{
			name: "basic pod creation",
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
			},
			agentConfig: &v1alpha1.AgentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-config",
				},
				Spec: v1alpha1.AgentConfigSpec{
					ServiceAccount: "test-sa",
					ConfigPVC:      "test-config-pvc",
					Provider:       "claude",
				},
			},
			image: "agentrun-runtime:latest",
			checkPod: func(pod *corev1.Pod) error {
				if pod.Name != "test-run-agent" {
					t.Errorf("Pod name = %v, want test-run-agent", pod.Name)
				}
				if pod.Namespace != "default" {
					t.Errorf("Pod namespace = %v, want default", pod.Namespace)
				}
				if pod.Spec.ServiceAccountName != "test-sa" {
					t.Errorf("ServiceAccount = %v, want test-sa", pod.Spec.ServiceAccountName)
				}
				if pod.Spec.RestartPolicy != corev1.RestartPolicyNever {
					t.Errorf("RestartPolicy = %v, want Never", pod.Spec.RestartPolicy)
				}
				// Check owner reference
				if len(pod.OwnerReferences) == 0 {
					t.Error("Pod has no owner references")
				}
				if pod.OwnerReferences[0].Name != "test-run" {
					t.Errorf("Owner reference name = %v, want test-run", pod.OwnerReferences[0].Name)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &Builder{
				Image: tt.image,
			}

			pod, err := builder.Build(tt.agentRun, tt.agentConfig)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

			if pod == nil {
				t.Fatal("Build() returned nil pod")
			}

			if err := tt.checkPod(pod); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestBuildSecurityContext(t *testing.T) {
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
	}

	agentConfig := &v1alpha1.AgentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-config",
		},
		Spec: v1alpha1.AgentConfigSpec{
			ServiceAccount: "test-sa",
			ConfigPVC:      "test-config-pvc",
			Provider:       "claude",
		},
	}

	builder := &Builder{
		Image: "agentrun-runtime:latest",
	}

	pod, err := builder.Build(agentRun, agentConfig)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Check pod security context
	if pod.Spec.SecurityContext == nil {
		t.Fatal("Pod SecurityContext is nil")
	}

	if pod.Spec.SecurityContext.RunAsNonRoot == nil || !*pod.Spec.SecurityContext.RunAsNonRoot {
		t.Error("RunAsNonRoot should be true")
	}

	if pod.Spec.SecurityContext.RunAsUser == nil || *pod.Spec.SecurityContext.RunAsUser != 65532 {
		t.Error("RunAsUser should be 65532")
	}

	if pod.Spec.SecurityContext.FSGroup == nil || *pod.Spec.SecurityContext.FSGroup != 65532 {
		t.Error("FSGroup should be 65532")
	}

	// Check container security context
	if len(pod.Spec.Containers) == 0 {
		t.Fatal("No containers in pod")
	}

	container := pod.Spec.Containers[0]
	if container.SecurityContext == nil {
		t.Fatal("Container SecurityContext is nil")
	}

	if container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
		t.Error("AllowPrivilegeEscalation should be false")
	}

	if container.SecurityContext.ReadOnlyRootFilesystem == nil || !*container.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("ReadOnlyRootFilesystem should be true")
	}

	if container.SecurityContext.Capabilities == nil || len(container.SecurityContext.Capabilities.Drop) == 0 {
		t.Error("Capabilities should drop ALL")
	}
}

func TestBuildVolumes(t *testing.T) {
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
	}

	agentConfig := &v1alpha1.AgentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-config",
		},
		Spec: v1alpha1.AgentConfigSpec{
			ServiceAccount: "test-sa",
			ConfigPVC:      "test-config-pvc",
			Provider:       "claude",
		},
	}

	builder := &Builder{
		Image: "agentrun-runtime:latest",
	}

	pod, err := builder.Build(agentRun, agentConfig)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have at least config volume
	if len(pod.Spec.Volumes) < 1 {
		t.Fatal("Pod should have at least 1 volume")
	}

	// Find config volume
	var configVolume *corev1.Volume
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == "config" {
			configVolume = &pod.Spec.Volumes[i]
			break
		}
	}

	if configVolume == nil {
		t.Fatal("Config volume not found")
	}

	if configVolume.PersistentVolumeClaim == nil {
		t.Fatal("Config volume should be PVC")
	}

	if configVolume.PersistentVolumeClaim.ClaimName != "test-config-pvc" {
		t.Errorf("Config PVC name = %v, want test-config-pvc", configVolume.PersistentVolumeClaim.ClaimName)
	}
}

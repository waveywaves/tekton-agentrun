package pod

import (
	"fmt"

	"github.com/waveywaves/agentrun-controller/pkg/apis/agent/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder builds Pod specs for agent execution
type Builder struct {
	Image string
}

// Build creates a Pod spec for an AgentRun
func (b *Builder) Build(agentRun *v1alpha1.AgentRun, agentConfig *v1alpha1.AgentConfig) (*corev1.Pod, error) {
	podName := fmt.Sprintf("%s-agent", agentRun.Name)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: agentRun.Namespace,
			Labels: map[string]string{
				"agent.tekton.dev/agentrun": agentRun.Name,
				"agent.tekton.dev/config":   agentConfig.Name,
				"app.kubernetes.io/component": "agent-runtime",
				"app.kubernetes.io/managed-by": "agentrun-controller",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(agentRun, v1alpha1.SchemeGroupVersion.WithKind("AgentRun")),
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: agentConfig.Spec.ServiceAccount,
			RestartPolicy:      corev1.RestartPolicyNever,
			SecurityContext:    b.buildPodSecurityContext(),
			Containers: []corev1.Container{
				{
					Name:            "agent",
					Image:           b.Image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					SecurityContext: b.buildContainerSecurityContext(),
					VolumeMounts:    b.buildVolumeMounts(),
					Env: []corev1.EnvVar{
						{
							Name:  "AGENTRUN_NAME",
							Value: agentRun.Name,
						},
						{
							Name:  "AGENTRUN_UID",
							Value: string(agentRun.UID),
						},
						{
							Name:  "AGENTRUN_NAMESPACE",
							Value: agentRun.Namespace,
						},
						{
							Name:  "AGENTRUN_GOAL",
							Value: agentRun.Spec.Goal,
						},
						{
							Name:  "AGENTCONFIG_NAME",
							Value: agentConfig.Name,
						},
						{
							Name:  "LLM_PROVIDER",
							Value: agentConfig.Spec.Provider,
						},
					},
				},
			},
			Volumes: b.buildVolumes(agentConfig),
		},
	}

	return pod, nil
}

func (b *Builder) buildPodSecurityContext() *corev1.PodSecurityContext {
	runAsNonRoot := true
	runAsUser := int64(65532)
	fsGroup := int64(65532)

	return &corev1.PodSecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &runAsUser,
		FSGroup:      &fsGroup,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func (b *Builder) buildContainerSecurityContext() *corev1.SecurityContext {
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := true
	runAsNonRoot := true

	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
		RunAsNonRoot:             &runAsNonRoot,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func (b *Builder) buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/workspace/config",
			ReadOnly:  true,
		},
		{
			Name:      "data",
			MountPath: "/workspace/data",
			ReadOnly:  false,
		},
		{
			Name:      "secrets",
			MountPath: "/workspace/secrets",
			ReadOnly:  true,
		},
	}
}

func (b *Builder) buildVolumes(agentConfig *v1alpha1.AgentConfig) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "agent-prompts",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "system.txt",
							Path: "prompts/system.txt",
						},
						{
							Key:  "planner.txt",
							Path: "prompts/planner.txt",
						},
						{
							Key:  "reflector.txt",
							Path: "prompts/reflector.txt",
						},
					},
				},
			},
		},
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "secrets",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "claude-api-key",
				},
			},
		},
	}
}

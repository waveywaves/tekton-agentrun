package agentrun

import (
	"context"
	"fmt"

	"github.com/waveywaves/agentrun-controller/pkg/apis/agent/v1alpha1"
	"github.com/waveywaves/agentrun-controller/pkg/pod"
	"github.com/waveywaves/agentrun-controller/pkg/security"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Reconciler reconciles AgentRun objects
type Reconciler struct {
	KubeClient   kubernetes.Interface
	Image        string
	AgentConfigs map[string]*v1alpha1.AgentConfig
}

// Reconcile handles the reconciliation of an AgentRun
func (r *Reconciler) Reconcile(ctx context.Context, agentRun *v1alpha1.AgentRun) error {
	// Check if already done
	if agentRun.IsDone() {
		return nil
	}

	// Get AgentConfig
	agentConfig, err := r.getAgentConfig(agentRun)
	if err != nil {
		return fmt.Errorf("failed to get AgentConfig: %w", err)
	}

	// Initialize start time if not set
	if !agentRun.HasStarted() {
		now := metav1.Now()
		agentRun.Status.StartTime = &now
	}

	// Handle based on current phase
	switch agentRun.Status.Phase {
	case v1alpha1.AgentRunPhasePending:
		return r.handlePending(ctx, agentRun, agentConfig)
	case v1alpha1.AgentRunPhaseActing:
		return r.handleActing(ctx, agentRun)
	default:
		// Unknown phase, set to Pending
		agentRun.Status.Phase = v1alpha1.AgentRunPhasePending
		return nil
	}
}

func (r *Reconciler) handlePending(ctx context.Context, agentRun *v1alpha1.AgentRun, agentConfig *v1alpha1.AgentConfig) error {
	// Create RBAC for agent pod
	if err := r.createRBAC(ctx, agentRun, agentConfig); err != nil {
		return fmt.Errorf("failed to create RBAC: %w", err)
	}

	// Create agent pod
	if err := r.createAgentPod(ctx, agentRun, agentConfig); err != nil {
		return fmt.Errorf("failed to create agent pod: %w", err)
	}

	// Update phase to Acting
	agentRun.Status.Phase = v1alpha1.AgentRunPhaseActing

	return nil
}

func (r *Reconciler) handleActing(ctx context.Context, agentRun *v1alpha1.AgentRun) error {
	// Get agent pod
	podName := fmt.Sprintf("%s-agent", agentRun.Name)
	agentPod, err := r.KubeClient.CoreV1().Pods(agentRun.Namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod doesn't exist, go back to Pending
			agentRun.Status.Phase = v1alpha1.AgentRunPhasePending
			return nil
		}
		return fmt.Errorf("failed to get agent pod: %w", err)
	}

	// Check pod status
	switch agentPod.Status.Phase {
	case corev1.PodSucceeded:
		// Agent completed successfully
		agentRun.Status.Phase = v1alpha1.AgentRunPhaseSucceeded
		now := metav1.Now()
		agentRun.Status.CompletionTime = &now
		return nil

	case corev1.PodFailed:
		// Agent failed
		agentRun.Status.Phase = v1alpha1.AgentRunPhaseFailed
		now := metav1.Now()
		agentRun.Status.CompletionTime = &now
		return nil

	default:
		// Still running, nothing to do
		return nil
	}
}

func (r *Reconciler) createRBAC(ctx context.Context, agentRun *v1alpha1.AgentRun, agentConfig *v1alpha1.AgentConfig) error {
	// Generate Role
	role := security.GenerateRole(agentRun)

	// Create Role
	_, err := r.KubeClient.RbacV1().Roles(agentRun.Namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create role: %w", err)
	}

	// Generate RoleBinding
	roleBinding := security.GenerateRoleBinding(agentRun, agentConfig, role.Name)

	// Create RoleBinding
	_, err = r.KubeClient.RbacV1().RoleBindings(agentRun.Namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create rolebinding: %w", err)
	}

	return nil
}

func (r *Reconciler) createAgentPod(ctx context.Context, agentRun *v1alpha1.AgentRun, agentConfig *v1alpha1.AgentConfig) error {
	// Build pod spec
	builder := &pod.Builder{
		Image: r.Image,
	}

	agentPod, err := builder.Build(agentRun, agentConfig)
	if err != nil {
		return fmt.Errorf("failed to build pod: %w", err)
	}

	// Create pod
	_, err = r.KubeClient.CoreV1().Pods(agentRun.Namespace).Create(ctx, agentPod, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	return nil
}

func (r *Reconciler) getAgentConfig(agentRun *v1alpha1.AgentRun) (*v1alpha1.AgentConfig, error) {
	// For now, use the in-memory map
	// In production, this would query the API server or use a lister
	config, ok := r.AgentConfigs[agentRun.Spec.ConfigRef.Name]
	if !ok {
		return nil, fmt.Errorf("AgentConfig %q not found", agentRun.Spec.ConfigRef.Name)
	}
	return config, nil
}

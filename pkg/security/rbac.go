package security

import (
	"fmt"

	"github.com/waveywaves/agentrun-controller/pkg/apis/agent/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateRole creates a read-only Role for an AgentRun
func GenerateRole(agentRun *v1alpha1.AgentRun) *rbacv1.Role {
	roleName := GenerateRoleName(agentRun)

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: agentRun.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(agentRun, v1alpha1.SchemeGroupVersion.WithKind("AgentRun")),
			},
			Labels: map[string]string{
				"agent.tekton.dev/agentrun": agentRun.Name,
				"app.kubernetes.io/component": "agent-rbac",
				"app.kubernetes.io/managed-by": "agentrun-controller",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				// Core resources - read only
				APIGroups: []string{""},
				Resources: []string{"pods", "services", "endpoints"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				// Apps resources - read only
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "replicasets", "statefulsets", "daemonsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				// Events - read only
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	return role
}

// GenerateRoleBinding creates a RoleBinding linking the Role to the ServiceAccount
func GenerateRoleBinding(agentRun *v1alpha1.AgentRun, agentConfig *v1alpha1.AgentConfig, roleName string) *rbacv1.RoleBinding {
	roleBindingName := GenerateRoleBindingName(agentRun)

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: agentRun.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(agentRun, v1alpha1.SchemeGroupVersion.WithKind("AgentRun")),
			},
			Labels: map[string]string{
				"agent.tekton.dev/agentrun": agentRun.Name,
				"app.kubernetes.io/component": "agent-rbac",
				"app.kubernetes.io/managed-by": "agentrun-controller",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      agentConfig.Spec.ServiceAccount,
				Namespace: agentRun.Namespace,
			},
		},
	}

	return rb
}

// GenerateRoleName generates a consistent Role name for an AgentRun
func GenerateRoleName(agentRun *v1alpha1.AgentRun) string {
	return fmt.Sprintf("agentrun-%s", agentRun.Name)
}

// GenerateRoleBindingName generates a consistent RoleBinding name for an AgentRun
func GenerateRoleBindingName(agentRun *v1alpha1.AgentRun) string {
	return fmt.Sprintf("agentrun-%s", agentRun.Name)
}

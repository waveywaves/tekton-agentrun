package security

import (
	"testing"

	"github.com/waveywaves/agentrun-controller/pkg/apis/agent/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateRole(t *testing.T) {
	tests := []struct {
		name      string
		agentRun  *v1alpha1.AgentRun
		wantRules int
		checkRule func(*rbacv1.PolicyRule) bool
	}{
		{
			name: "generate role with read-only permissions",
			agentRun: &v1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
			},
			wantRules: 3, // core resources, apps resources, events
			checkRule: func(rule *rbacv1.PolicyRule) bool {
				// Verify all verbs are read-only
				for _, verb := range rule.Verbs {
					if verb != "get" && verb != "list" && verb != "watch" {
						return false
					}
				}
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := GenerateRole(tt.agentRun)

			if role == nil {
				t.Fatal("GenerateRole() returned nil")
			}

			if role.Name == "" {
				t.Error("Role name is empty")
			}

			if role.Namespace != tt.agentRun.Namespace {
				t.Errorf("Role namespace = %v, want %v", role.Namespace, tt.agentRun.Namespace)
			}

			if len(role.Rules) != tt.wantRules {
				t.Errorf("Role has %d rules, want %d", len(role.Rules), tt.wantRules)
			}

			for i, rule := range role.Rules {
				if !tt.checkRule(&rule) {
					t.Errorf("Rule %d failed validation: %+v", i, rule)
				}
			}

			// Verify owner reference
			if len(role.OwnerReferences) == 0 {
				t.Error("Role has no owner references")
			}
		})
	}
}

func TestGenerateRoleBinding(t *testing.T) {
	tests := []struct {
		name           string
		agentRun       *v1alpha1.AgentRun
		agentConfig    *v1alpha1.AgentConfig
		roleName       string
		wantSAName     string
		wantNamespace  string
	}{
		{
			name: "generate role binding",
			agentRun: &v1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
			},
			agentConfig: &v1alpha1.AgentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-config",
				},
				Spec: v1alpha1.AgentConfigSpec{
					ServiceAccount: "test-sa",
					ConfigPVC:      "test-pvc",
				},
			},
			roleName:      "test-role",
			wantSAName:    "test-sa",
			wantNamespace: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := GenerateRoleBinding(tt.agentRun, tt.agentConfig, tt.roleName)

			if rb == nil {
				t.Fatal("GenerateRoleBinding() returned nil")
			}

			if rb.Name == "" {
				t.Error("RoleBinding name is empty")
			}

			if rb.Namespace != tt.wantNamespace {
				t.Errorf("RoleBinding namespace = %v, want %v", rb.Namespace, tt.wantNamespace)
			}

			if rb.RoleRef.Name != tt.roleName {
				t.Errorf("RoleRef.Name = %v, want %v", rb.RoleRef.Name, tt.roleName)
			}

			if rb.RoleRef.Kind != "Role" {
				t.Errorf("RoleRef.Kind = %v, want Role", rb.RoleRef.Kind)
			}

			if len(rb.Subjects) != 1 {
				t.Fatalf("RoleBinding has %d subjects, want 1", len(rb.Subjects))
			}

			if rb.Subjects[0].Name != tt.wantSAName {
				t.Errorf("Subject name = %v, want %v", rb.Subjects[0].Name, tt.wantSAName)
			}

			if rb.Subjects[0].Kind != "ServiceAccount" {
				t.Errorf("Subject kind = %v, want ServiceAccount", rb.Subjects[0].Kind)
			}

			// Verify owner reference
			if len(rb.OwnerReferences) == 0 {
				t.Error("RoleBinding has no owner references")
			}
		})
	}
}

func TestGenerateRoleName(t *testing.T) {
	tests := []struct {
		name     string
		agentRun *v1alpha1.AgentRun
		want     string
	}{
		{
			name: "generate role name from agentrun",
			agentRun: &v1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
			},
			want: "agentrun-test-run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateRoleName(tt.agentRun)
			if got != tt.want {
				t.Errorf("GenerateRoleName() = %v, want %v", got, tt.want)
			}
		})
	}
}

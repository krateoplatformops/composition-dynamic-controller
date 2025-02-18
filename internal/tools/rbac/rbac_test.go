package rbac

import (
	"testing"
)

func TestInitRole(t *testing.T) {
	name := "test-role"
	namespace := "default"
	role := InitRole(name, namespace)

	if role.ObjectMeta.Name != name {
		t.Errorf("expected role name %s, got %s", name, role.ObjectMeta.Name)
	}

	if len(role.Rules) != 0 {
		t.Errorf("expected role rules to be empty, got %d rules", len(role.Rules))
	}
}

func TestInitRoleBinding(t *testing.T) {
	name := "test-rolebinding"
	roleRefName := "test-role"
	namespace := "default"
	saName := "test-sa"
	saNamespace := "default"
	roleBinding := InitRoleBinding(name, roleRefName, namespace, saName, saNamespace)

	if roleBinding.ObjectMeta.Name != name {
		t.Errorf("expected role binding name %s, got %s", name, roleBinding.ObjectMeta.Name)
	}

	if roleBinding.RoleRef.Name != roleRefName {
		t.Errorf("expected role ref name %s, got %s", roleRefName, roleBinding.RoleRef.Name)
	}

	if len(roleBinding.Subjects) != 1 {
		t.Errorf("expected 1 subject, got %d", len(roleBinding.Subjects))
	}

	subject := roleBinding.Subjects[0]
	if subject.Kind != "ServiceAccount" || subject.Name != saName || subject.Namespace != saNamespace {
		t.Errorf("expected subject to be ServiceAccount %s in namespace %s, got %s %s in namespace %s", saName, saNamespace, subject.Kind, subject.Name, subject.Namespace)
	}
}

func TestInitClusterRole(t *testing.T) {
	name := "test-clusterrole"
	clusterRole := InitClusterRole(name)

	if clusterRole.ObjectMeta.Name != name {
		t.Errorf("expected cluster role name %s, got %s", name, clusterRole.ObjectMeta.Name)
	}

	if len(clusterRole.Rules) != 0 {
		t.Errorf("expected cluster role rules to be empty, got %d rules", len(clusterRole.Rules))
	}
}

func TestInitClusterRoleBinding(t *testing.T) {
	name := "test-clusterrolebinding"
	roleRefName := "test-clusterrole"
	saName := "test-sa"
	saNamespace := "default"
	clusterRoleBinding := InitClusterRoleBinding(name, roleRefName, saName, saNamespace)

	if clusterRoleBinding.ObjectMeta.Name != name {
		t.Errorf("expected cluster role binding name %s, got %s", name, clusterRoleBinding.ObjectMeta.Name)
	}

	if clusterRoleBinding.RoleRef.Name != roleRefName {
		t.Errorf("expected role ref name %s, got %s", roleRefName, clusterRoleBinding.RoleRef.Name)
	}

	if len(clusterRoleBinding.Subjects) != 1 {
		t.Errorf("expected 1 subject, got %d", len(clusterRoleBinding.Subjects))
	}

	subject := clusterRoleBinding.Subjects[0]
	if subject.Kind != "ServiceAccount" || subject.Name != saName || subject.Namespace != saNamespace {
		t.Errorf("expected subject to be ServiceAccount %s in namespace %s, got %s %s in namespace %s", saName, saNamespace, subject.Kind, subject.Name, subject.Namespace)
	}
}
